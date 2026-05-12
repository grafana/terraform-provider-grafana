package cloud

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/grafana/grafana-com-public-clients/go/gcom"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
)

const (
	// createStackGetInstanceExistsBudget bounds retries while waiting for the stack record to show up (HTTP 404 → 200).
	createStackGetInstanceExistsBudget = 1 * time.Minute
	// createStackGetInstanceTransientBudget bounds retries for HTTP 429 and 5xx from stack creation start.
	createStackGetInstanceTransientBudget = 2 * time.Minute
)

func stackCreateLaterDeadline(a, b time.Time) time.Time {
	if a.After(b) {
		return a
	}
	return b
}

// stackCreateGetInstanceRetryDecision implements deadline rules for getInstanceWithCreateStackRetries:
//   - HTTP 404 is retried only when retryNotFound is true, until createStackGetInstanceExistsBudget elapses from start.
//   - HTTP 429 and 5xx extend the overall deadline to at least createStackGetInstanceTransientBudget from start.
func stackCreateGetInstanceRetryDecision(
	retryNotFound bool,
	start, now, effectiveDeadline time.Time,
	resp *http.Response,
) (retry bool, newDeadline time.Time) {
	newDeadline = effectiveDeadline
	if resp == nil {
		return false, newDeadline
	}

	switch resp.StatusCode {
	case http.StatusNotFound:
		retry = retryNotFound && now.Before(start.Add(createStackGetInstanceExistsBudget))
	case http.StatusTooManyRequests:
		newDeadline = stackCreateLaterDeadline(effectiveDeadline, start.Add(createStackGetInstanceTransientBudget))
		retry = now.Before(newDeadline)
	default:
		if resp.StatusCode >= http.StatusInternalServerError && resp.StatusCode < 600 {
			newDeadline = stackCreateLaterDeadline(effectiveDeadline, start.Add(createStackGetInstanceTransientBudget))
			retry = now.Before(newDeadline)
		}
	}
	return retry, newDeadline
}

// stackCreateGetInstanceRetryStrategy builds a RetryStrategy for RetryAPIRequest from stackCreateGetInstanceRetryDecision.
func stackCreateGetInstanceRetryStrategy(retryNotFound bool, start time.Time, effectiveDeadline *time.Time) RetryStrategy {
	return func(err error, resp *http.Response) *retry.RetryError {
		if resp != nil && resp.StatusCode >= http.StatusOK && resp.StatusCode < http.StatusMultipleChoices {
			if err != nil {
				return retry.NonRetryableError(err)
			}
			return nil
		}

		now := time.Now()
		doRetry, newDeadline := stackCreateGetInstanceRetryDecision(retryNotFound, start, now, *effectiveDeadline, resp)
		*effectiveDeadline = newDeadline

		if !doRetry {
			return retry.NonRetryableError(httpAttemptError(err, resp))
		}
		return retry.RetryableError(httpAttemptError(err, resp))
	}
}

// getInstanceWithCreateStackRetries wraps InstancesAPI.GetInstance for CreateStackWithRetries using RetryAPIRequest:
// it retries HTTP 404 (when retryNotFound) for up to 1 minute from the first attempt, and retries HTTP 429 / 5xx for up to
// 2 minutes from the first attempt (extending the inner deadline when those statuses appear).
func getInstanceWithCreateStackRetries(
	ctx context.Context,
	client *gcom.APIClient,
	slug string,
	retryNotFound bool,
) (*gcom.FormattedApiInstance, *http.Response, error) {
	start := time.Now()
	effectiveDeadline := start.Add(createStackGetInstanceExistsBudget)
	strategy := stackCreateGetInstanceRetryStrategy(retryNotFound, start, &effectiveDeadline)

	var captured *gcom.FormattedApiInstance
	var lastResp *http.Response

	apiErr := RetryAPIRequest(ctx, createStackGetInstanceTransientBudget, defaultRetryPollInterval, strategy, func() (*http.Response, error) {
		inst, resp, execErr := client.InstancesAPI.GetInstance(ctx, slug).Execute()
		lastResp = resp
		if execErr == nil {
			captured = inst
		}
		return resp, execErr
	})
	if apiErr != nil {
		return captured, lastResp, fmt.Errorf("GetInstance %q during stack create: %w", slug, apiErr)
	}
	return captured, lastResp, nil
}
