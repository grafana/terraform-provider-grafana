package cloud

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/grafana/grafana-com-public-clients/go/gcom"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
)

// needsAnotherCreateStackHTTPAttempt reports whether CreateStackV1 should immediately receive another HTTP attempt
// (inside RetryAPIRequest): Grafana Cloud rate limiting or upstream/server faults during provisioning.
// Slug conflicts, validation failures, and other client-visible errors do not use extra attempts here.
func needsAnotherCreateStackHTTPAttempt(resp *http.Response) bool {
	if resp == nil {
		return false
	}
	if resp.StatusCode == http.StatusTooManyRequests {
		return true
	}
	return resp.StatusCode >= http.StatusInternalServerError && resp.StatusCode < 600
}

// CreateStackRetryStrategy is the RetryStrategy for each inner RetryAPIRequest pass over CreateStackV1 HTTP attempts.
var CreateStackRetryStrategy RetryStrategy = func(err error, resp *http.Response) *retry.RetryError {
	if resp == nil {
		if err == nil {
			return nil
		}
		return retry.NonRetryableError(err)
	}
	if resp.StatusCode >= http.StatusOK && resp.StatusCode < http.StatusMultipleChoices {
		if err != nil {
			return retry.NonRetryableError(err)
		}
		return nil
	}
	ae := httpAttemptError(err, resp)
	if needsAnotherCreateStackHTTPAttempt(resp) {
		return retry.RetryableError(ae)
	}
	return retry.NonRetryableError(ae)
}

// CreateStackOuterRetryDecision is the outcome after RetryAPIRequest returns an error for CreateStackV1.
type CreateStackOuterRetryDecision struct {
	StopWithErr         error
	AdoptedInstance     *gcom.FormattedApiInstance
	SleepBeforeContinue time.Duration
}

// errorWithResponseBody matches errors that expose raw HTTP body bytes (e.g. *gcom.GenericOpenAPIError).
type errorWithResponseBody interface {
	error
	Body() []byte
}

func graceSleepForTransient409Conflict(execErr error, slug string) (time.Duration, bool) {
	var bodyErr errorWithResponseBody
	if !errors.As(execErr, &bodyErr) {
		return 0, false
	}
	body := string(bodyErr.Body())
	if !strings.Contains(body, "deleted recently") && !strings.Contains(body, "Grafana stack with the same slug already exists") {
		return 0, false
	}
	waitTime := 35 * time.Second
	gracePeriodRe := regexp.MustCompile(`wait for (\d+)s`)
	if matches := gracePeriodRe.FindSubmatch(bodyErr.Body()); len(matches) == 2 {
		if seconds, parseErr := strconv.Atoi(string(matches[1])); parseErr == nil {
			log.Printf("[WARN] slug %s is temporarily unavailable, retrying after %s", slug, waitTime)
			waitTime = time.Duration(seconds+5) * time.Second // +5s margin for clock skew
		}
	}
	return waitTime, true
}

// DecideCreateStackOuterRetry applies conflict handling, adoption reads, and outer-loop backoff after CreateStackV1
// RetryAPIRequest fails (non-nil apiErr). It corresponds to the former outer create-loop switch body.
func DecideCreateStackOuterRetry(
	ctx context.Context,
	slug string,
	apiErr, lastExecErr error,
	lastResp *http.Response,
	getInstance func(context.Context, string) (*gcom.FormattedApiInstance, error),
) CreateStackOuterRetryDecision {
	switch {
	case lastExecErr != nil && lastResp != nil && lastResp.StatusCode == http.StatusConflict:
		// 409 Conflict — the slug is unavailable. GCOM returns this for several reasons:
		//   1. A stack with the same slug is still active (real conflict).
		//   2. A stack with the same slug was deleted within the GCOM grace period (~30s).
		//   3. Downstream systems (stack-state-service, hosted-grafana) haven't finished cleanup.
		// Only case 2 is transient and worth retrying — the GCOM message contains
		// "deleted recently" with the grace period duration. Cases 1 and 3 are genuine
		// conflicts that won't resolve by waiting; surface those to the Terraform user.
		if wait, ok := graceSleepForTransient409Conflict(lastExecErr, slug); ok {
			return CreateStackOuterRetryDecision{SleepBeforeContinue: wait}
		}
		existing, err := getInstance(ctx, slug)
		if err == nil && existing != nil && existing.Status != "deleted" {
			return CreateStackOuterRetryDecision{
				StopWithErr: fmt.Errorf(
					"cannot create Grafana Cloud stack: slug %q is already used by an existing stack (id %v)",
					slug, existing.Id,
				),
			}
		}
		return CreateStackOuterRetryDecision{StopWithErr: lastExecErr}
	case lastExecErr != nil:
		// If we had an error that isn't a conflict (already exists), try to read the stack.
		// Sometimes the stack is created but the API returns an error (e.g. 504).
		adopted, err := getInstance(ctx, slug)
		if err == nil {
			return CreateStackOuterRetryDecision{AdoptedInstance: adopted}
		}
		return CreateStackOuterRetryDecision{SleepBeforeContinue: 10 * time.Second}
	default:
		return CreateStackOuterRetryDecision{StopWithErr: apiErr}
	}
}
