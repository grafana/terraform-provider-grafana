package cloud

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
)

// RetryStrategy interprets one HTTP attempt (before the response body is consumed
// by RetryAPIRequest on retry paths).
//
// Return nil when the operation succeeded and no further attempts are needed.
// Return retry.RetryableError(err) to schedule another attempt after Retry-After
// (if present on the response) or PollInterval.
// Return retry.NonRetryableError(err) to stop immediately and propagate err.
type RetryStrategy func(err error, resp *http.Response) *retry.RetryError

// GetRetryStrategy returns a RetryStrategy suitable for typical Grafana Cloud REST calls:
//   - HTTP 5xx responses are retried (wait uses PollInterval or Retry-After when present).
//   - HTTP 429 responses are retried; Retry-After is interpreted by RetryAPIRequest when present.
//   - HTTP 404 responses are not retried.
//   - Other HTTP responses use err when non-nil, otherwise a status-derived error, and are not retried.
//   - When resp is nil, only the transport error err is considered and it is not retried.
//
// Success is HTTP 2xx with err == nil.
func GetRetryStrategy() RetryStrategy {
	return func(err error, resp *http.Response) *retry.RetryError {
		if resp == nil {
			if err == nil {
				return nil
			}
			return retry.NonRetryableError(err)
		}

		code := resp.StatusCode
		switch {
		case code == http.StatusNotFound:
			return retry.NonRetryableError(httpAttemptError(err, resp))
		case code == http.StatusTooManyRequests:
			return retry.RetryableError(httpAttemptError(err, resp))
		case code >= http.StatusInternalServerError && code < 600:
			return retry.RetryableError(httpAttemptError(err, resp))
		case code >= http.StatusOK && code < http.StatusMultipleChoices:
			if err != nil {
				return retry.NonRetryableError(err)
			}
			return nil
		default:
			return retry.NonRetryableError(httpAttemptError(err, resp))
		}
	}
}

func httpAttemptError(err error, resp *http.Response) error {
	if err != nil {
		return err
	}
	return fmt.Errorf("HTTP %s", resp.Status)
}

const defaultRetryPollInterval = 500 * time.Millisecond

// RetryAPIRequest executes fn until strategy returns nil, ctx is cancelled,
// timeout elapses, or strategy returns a non-retry error.
//
// pollInterval is the wait between attempts when the response has no valid
// Retry-After header (or no response). Values <= 0 default to 500ms, matching
// RetryContext's MinTimeout.
//
// When strategy signals a retry, RetryAPIRequest drains and closes resp.Body
// before waiting so callers should read the body inside fn before returning if
// they need it, or only after RetryAPIRequest returns nil without retrying.
func RetryAPIRequest(ctx context.Context, timeout, pollInterval time.Duration, strategy RetryStrategy, fn func() (*http.Response, error)) error {
	if strategy == nil {
		return errors.New("RetryAPIRequest: strategy is nil")
	}
	if fn == nil {
		return errors.New("RetryAPIRequest: fn is nil")
	}

	waitFallback := pollInterval
	if waitFallback <= 0 {
		waitFallback = defaultRetryPollInterval
	}

	deadline := time.Now().Add(timeout)
	var lastAttemptErr error

	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		if time.Now().After(deadline) {
			if lastAttemptErr != nil {
				return fmt.Errorf("retry API request: timeout after %s: %w", timeout, lastAttemptErr)
			}
			return fmt.Errorf("retry API request: timeout after %s", timeout)
		}

		resp, err := fn()
		decision := strategy(err, resp)

		if decision == nil {
			return nil
		}
		if !decision.Retryable {
			return decision.Err
		}

		lastAttemptErr = decision.Err
		drainAndCloseResponse(resp)

		wait := waitFallback
		if d, ok := parseRetryAfter(resp); ok {
			wait = d
		}
		if wait < 0 {
			wait = 0
		}

		remaining := time.Until(deadline)
		if remaining <= 0 {
			return fmt.Errorf("retry API request: timeout after %s: %w", timeout, lastAttemptErr)
		}
		if wait > remaining {
			wait = remaining
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(wait):
		}
	}
}

func drainAndCloseResponse(resp *http.Response) {
	if resp == nil || resp.Body == nil {
		return
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	_ = resp.Body.Close()
}

// parseRetryAfter returns the delay indicated by Retry-After per RFC 7231
// (delta-seconds or HTTP-date).
func parseRetryAfter(resp *http.Response) (time.Duration, bool) {
	if resp == nil {
		return 0, false
	}
	v := strings.TrimSpace(resp.Header.Get("Retry-After"))
	if v == "" {
		return 0, false
	}
	if secs, err := strconv.Atoi(v); err == nil && secs >= 0 {
		return time.Duration(secs) * time.Second, true
	}
	if t, err := http.ParseTime(v); err == nil {
		d := time.Until(t)
		if d < 0 {
			return 0, true
		}
		return d, true
	}
	return 0, false
}
