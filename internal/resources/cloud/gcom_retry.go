package cloud

import (
	"context"
	"errors"
	"fmt"
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
	// DefaultHTTPRequestRetryTimeout caps the HTTP request retry loop duration.
	DefaultHTTPRequestRetryTimeout = 2 * time.Minute
	defaultHTTPRetryWait           = 1 * time.Second
	httpRetryAfterCap              = 2 * time.Minute
)

// DefaultHTTPRequestRetryConfig returns the standard Grafana Cloud HTTP retry policy.
func DefaultHTTPRequestRetryConfig() HTTPRequestRetryConfig {
	return HTTPRequestRetryConfig{
		Timeout:                 DefaultHTTPRequestRetryTimeout,
		TransientErrorAnalyzers: []TransientErrorAnalyzer{DefaultGCOMTransient},
		RetryWait:               DefaultHTTPRetryWait,
	}
}

// ErrorAnalyzer can reinterpret an operation error before retry classification.
type ErrorAnalyzer func(resp *http.Response, err error) error

// TransientErrorAnalyzer reports whether an operation error should be retried.
type TransientErrorAnalyzer func(resp *http.Response, err error) bool

// RetryWait returns the delay before the next retry attempt.
type RetryWait func(resp *http.Response, err error) (time.Duration, bool)

// HTTPRequestRetryConfig configures RetryHTTPRequest.
type HTTPRequestRetryConfig struct {
	Timeout time.Duration

	// ErrorAnalyzer, when non-nil, can transform or accept errors before retry classification.
	ErrorAnalyzer ErrorAnalyzer

	// TransientErrorAnalyzers classify retryable errors.
	TransientErrorAnalyzers []TransientErrorAnalyzer

	// RetryWait determines how long to wait before the next retry.
	RetryWait RetryWait
}

// RetryHTTPRequest runs op under retry.RetryContext until success or non-retryable error.
//
// Follow-up retries:
//   - Sleeps according to RetryWait and ctx cancellation. The default uses Retry-After (RFC 9110 delta-seconds or HTTP-date)
//     for HTTP 429 responses and a short fallback wait for other retryable errors, capped by httpRetryAfterCap.
//     If the requested wait exceeds the remaining retry budget, the error is returned immediately instead of sleeping and timing out anyway
//     (grafana.com rate limit windows can be up to an hour, far beyond the retry timeout).
//   - ErrorAnalyzer can accept or transform errors before they are classified for retry.
//   - Failed response bodies are drained for connection reuse (success responses are unchanged).
func RetryHTTPRequest(ctx context.Context, cfg HTTPRequestRetryConfig, op func() (*http.Response, error)) error {
	timeout := cfg.Timeout
	deadline := time.Now().Add(timeout)

	return retry.RetryContext(ctx, timeout, func() *retry.RetryError {
		resp, err := op()
		if err == nil {
			return nil
		}

		if cfg.ErrorAnalyzer != nil {
			err = cfg.ErrorAnalyzer(resp, err)
			if err == nil {
				drainResponse(resp)
				return nil
			}
		}

		if transientHTTPError(cfg, resp, err) {
			logHTTPRequestRetryWarning(resp, err)
			if wait, ok := retryWait(cfg, resp, err); ok {
				if wait > time.Until(deadline) {
					drainResponse(resp)
					return retry.NonRetryableError(fmt.Errorf("retry wait (%s) exceeds the remaining retry budget, giving up: %w", wait, err))
				}
				sleepRetry(ctx, wait)
			}
			drainResponse(resp)
			return retry.RetryableError(err)
		}

		drainResponse(resp)
		return retry.NonRetryableError(err)
	})
}

func transientHTTPError(cfg HTTPRequestRetryConfig, resp *http.Response, err error) bool {
	if err == nil {
		return false
	}

	analyzers := cfg.TransientErrorAnalyzers
	for _, analyzer := range analyzers {
		if analyzer != nil && analyzer(resp, err) {
			return true
		}
	}
	return false
}

func retryWait(cfg HTTPRequestRetryConfig, resp *http.Response, err error) (time.Duration, bool) {
	if cfg.RetryWait != nil {
		return cfg.RetryWait(resp, err)
	}
	return 0, false
}

// AcceptNotFounds treats HTTP 404 responses as success, typically for idempotent DELETE calls.
func AcceptNotFounds(resp *http.Response, err error) error {
	if err != nil && resp != nil && resp.StatusCode == http.StatusNotFound {
		return nil
	}
	return err
}

// DefaultHTTPRetryWait returns the default delay before retrying transient HTTP request errors.
func DefaultHTTPRetryWait(resp *http.Response, err error) (time.Duration, bool) {
	if err == nil {
		return 0, false
	}
	if resp != nil && resp.StatusCode == http.StatusTooManyRequests {
		if wait, ok := parseRetryAfter(resp.Header.Get("Retry-After"), time.Now()); ok {
			return wait, true
		}
	}
	return defaultHTTPRetryWait, true
}

// DefaultGCOMTransient classifies Grafana Cloud transient HTTP responses and transport errors.
//
// Treats typical 429/408/5xx as retryable. Does not treat 409 as retryable — stack creation keeps bespoke conflict handling (see resource_cloud_stack createStack).
//
// Errors satisfying [net.Error] (including *[net.OpError]) are retryable unless the chain includes [context.DeadlineExceeded] (explicit client deadline / cancellation).
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

	if errors.Is(err, context.DeadlineExceeded) {
		return false
	}

	var netErr net.Error
	if errors.As(err, &netErr) {
		return true
	}

	return errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF)
}

func drainResponse(resp *http.Response) {
	if resp == nil || resp.Body == nil {
		return
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	_ = resp.Body.Close()
}

// sleepRetry sleeps before the next retry attempt.
func sleepRetry(ctx context.Context, d time.Duration) {
	if d <= 0 {
		return
	}
	if d > httpRetryAfterCap {
		d = httpRetryAfterCap
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

func logHTTPRequestRetryWarning(resp *http.Response, err error) {
	if err == nil {
		return
	}
	status := 0
	if resp != nil {
		status = resp.StatusCode
	}
	log.Printf("[WARN] Grafana Cloud API: retrying after transient error (HTTP status=%d): %v", status, err)
}
