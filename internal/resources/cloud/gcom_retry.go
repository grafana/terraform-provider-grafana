package cloud

import (
	"context"
	"errors"
	"io"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
)

const (
	// DefaultGCOMRetryTimeout caps the Grafana Cloud OpenAPI retry loop duration.
	DefaultGCOMRetryTimeout = 2 * time.Minute
	gcomRetryAfterCap       = 2 * time.Minute
)

// GCOMRetryConfig configures RetryGCOM.
type GCOMRetryConfig struct {
	Timeout time.Duration

	// TreatNotFoundAsSuccess, when true, treats HTTP 404 responses as success (typically for DELETE),
	// so destroys are idempotent when the resource is already gone.
	TreatNotFoundAsSuccess bool

	// OnTransient, when non-nil, is consulted after DefaultGCOMTransient returns false.
	OnTransient func(resp *http.Response, err error) bool
}

func (c GCOMRetryConfig) timeoutDuration() time.Duration {
	if c.Timeout > 0 {
		return c.Timeout
	}
	return DefaultGCOMRetryTimeout
}

// RetryGCOM runs op under retry.RetryContext until success or non-retryable error.
//
// Follow-up retries:
// - For HTTP 429, sleeps according to Retry-After (RFC 9110 delta-seconds or HTTP-date), capped by gcomRetryAfterCap and ctx cancellation (via SleepRetryAfterHeader).
// - When TreatNotFoundAsSuccess is set, HTTP 404 is treated as immediate success (common for idempotent DELETE).
// - Failed response bodies are drained for connection reuse (success responses are unchanged).
func RetryGCOM(ctx context.Context, cfg GCOMRetryConfig, op func() (*http.Response, error)) error {
	timeout := cfg.timeoutDuration()

	return retry.RetryContext(ctx, timeout, func() *retry.RetryError {
		resp, err := op()
		if err == nil {
			return nil
		}

		if cfg.TreatNotFoundAsSuccess && resp != nil && resp.StatusCode == http.StatusNotFound {
			gcomDrainResponse(resp)
			return nil
		}

		if transientGCOMError(cfg, resp, err) {
			logGCOMRetryWarning(resp, err)
			if resp != nil && resp.StatusCode == http.StatusTooManyRequests {
				SleepRetryAfterHeader(ctx, resp.Header.Get("Retry-After"))
			}
			gcomDrainResponse(resp)
			return retry.RetryableError(err)
		}

		gcomDrainResponse(resp)
		return retry.NonRetryableError(err)
	})
}

func transientGCOMError(cfg GCOMRetryConfig, resp *http.Response, err error) bool {
	if err == nil {
		return false
	}
	if DefaultGCOMTransient(resp, err) {
		return true
	}
	if cfg.OnTransient != nil && cfg.OnTransient(resp, err) {
		return true
	}
	return false
}

// DefaultGCOMTransient classifies Grafana Cloud transient HTTP responses and transport errors.
//
// Treats typical 429/408/5xx as retryable. Does not treat 409 as retryable — stack creation keeps bespoke conflict handling (see resource_cloud_stack createStack).
func DefaultGCOMTransient(resp *http.Response, err error) bool {
	if err == nil {
		return false
	}

	if resp != nil {
		switch resp.StatusCode {
		case http.StatusTooManyRequests, http.StatusRequestTimeout:
			return true
		case http.StatusInternalServerError, http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
			return true
		default:
			return resp.StatusCode >= 500
		}
	}

	var netErr net.Error
	if errors.As(err, &netErr) && (netErr.Timeout() || netErr.Temporary()) {
		return true
	}

	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
		return true
	}

	if errors.Is(err, context.DeadlineExceeded) {
		return false
	}

	var opErr *net.OpError
	return errors.As(err, &opErr)
}

func gcomDrainResponse(resp *http.Response) {
	if resp == nil || resp.Body == nil {
		return
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	_ = resp.Body.Close()
}

// SleepRetryAfterHeader sleeps per the Retry-After header value before the next retry attempt.
func SleepRetryAfterHeader(ctx context.Context, header string) {
	d, ok := parseRetryAfter(header, time.Now())
	if !ok || d <= 0 {
		return
	}
	if d > gcomRetryAfterCap {
		d = gcomRetryAfterCap
	}
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return
	case <-timer.C:
	}
}

// parseRetryAfter returns the wait duration from Retry-After (RFC 9110).
func parseRetryAfter(header string, now time.Time) (time.Duration, bool) {
	header = strings.TrimSpace(header)
	if header == "" {
		return 0, false
	}

	if secs, err := strconv.Atoi(header); err == nil && secs >= 0 {
		return time.Duration(secs) * time.Second, true
	}

	if t, err := http.ParseTime(header); err == nil {
		wait := t.Sub(now)
		if wait < 0 {
			return 0, true
		}
		return wait, true
	}

	return 0, false
}

func logGCOMRetryWarning(resp *http.Response, err error) {
	if err == nil {
		return
	}
	status := 0
	if resp != nil {
		status = resp.StatusCode
	}
	log.Printf("[WARN] Grafana Cloud API: retrying after transient error (HTTP status=%d): %v", status, err)
}
