package cloud_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/grafana/terraform-provider-grafana/v4/internal/resources/cloud"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
)

func TestRetryAPIRequest_nilStrategy(t *testing.T) {
	t.Parallel()
	err := cloud.RetryAPIRequest(context.Background(), time.Second, time.Millisecond, nil, func() (*http.Response, error) {
		return nil, nil
	})
	if err == nil || err.Error() != "RetryAPIRequest: strategy is nil" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRetryAPIRequest_nilFn(t *testing.T) {
	t.Parallel()
	err := cloud.RetryAPIRequest(context.Background(), time.Second, time.Millisecond, func(error, *http.Response) *retry.RetryError {
		return nil
	}, nil)
	if err == nil || err.Error() != "RetryAPIRequest: fn is nil" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRetryAPIRequest_successFirstAttempt(t *testing.T) {
	t.Parallel()
	var calls atomic.Int32
	err := cloud.RetryAPIRequest(context.Background(), time.Second, time.Millisecond,
		func(err error, resp *http.Response) *retry.RetryError {
			if err != nil {
				return retry.NonRetryableError(err)
			}
			if resp.StatusCode != http.StatusOK {
				return retry.RetryableError(fmt.Errorf("status %d", resp.StatusCode))
			}
			return nil
		},
		func() (*http.Response, error) {
			calls.Add(1)
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader("")),
			}, nil
		})
	if err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 1 {
		t.Fatalf("expected 1 call, got %d", calls.Load())
	}
}

func TestRetryAPIRequest_retryThenSuccess(t *testing.T) {
	t.Parallel()
	var calls atomic.Int32
	err := cloud.RetryAPIRequest(context.Background(), time.Second, 10*time.Millisecond,
		func(err error, resp *http.Response) *retry.RetryError {
			if err != nil {
				return retry.NonRetryableError(err)
			}
			if resp.StatusCode == http.StatusTooManyRequests {
				return retry.RetryableError(errors.New("too many requests"))
			}
			if resp.StatusCode != http.StatusOK {
				return retry.NonRetryableError(fmt.Errorf("unexpected status %d", resp.StatusCode))
			}
			return nil
		},
		func() (*http.Response, error) {
			n := calls.Add(1)
			if n == 1 {
				return &http.Response{
					StatusCode: http.StatusTooManyRequests,
					Body:       io.NopCloser(strings.NewReader("")),
				}, nil
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader("")),
			}, nil
		})
	if err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 2 {
		t.Fatalf("expected 2 calls, got %d", calls.Load())
	}
}

func TestRetryAPIRequest_nonRetryable(t *testing.T) {
	t.Parallel()
	apiErr := errors.New("bad request")
	err := cloud.RetryAPIRequest(context.Background(), time.Second, time.Millisecond,
		func(err error, resp *http.Response) *retry.RetryError {
			if err != nil {
				return retry.NonRetryableError(err)
			}
			if resp.StatusCode == http.StatusBadRequest {
				return retry.NonRetryableError(fmt.Errorf("not retryable: %w", apiErr))
			}
			return nil
		},
		func() (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusBadRequest,
				Body:       io.NopCloser(strings.NewReader("")),
			}, nil
		})
	if err == nil || !errors.Is(err, apiErr) {
		t.Fatalf("expected wrapped apiErr, got %v", err)
	}
}

func TestRetryAPIRequest_timeout(t *testing.T) {
	t.Parallel()
	err := cloud.RetryAPIRequest(context.Background(), 80*time.Millisecond, 15*time.Millisecond,
		func(err error, resp *http.Response) *retry.RetryError {
			if err != nil {
				return retry.NonRetryableError(err)
			}
			return retry.RetryableError(errors.New("retry"))
		},
		func() (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusServiceUnavailable,
				Body:       io.NopCloser(strings.NewReader("")),
			}, nil
		})
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if want := "retry API request: timeout"; !strings.Contains(err.Error(), want) {
		t.Fatalf("expected error containing %q, got %v", want, err)
	}
}

func TestRetryAPIRequest_contextCancel(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	var calls atomic.Int32
	err := cloud.RetryAPIRequest(ctx, time.Minute, 30*time.Millisecond,
		func(err error, resp *http.Response) *retry.RetryError {
			if err != nil {
				return retry.NonRetryableError(err)
			}
			return retry.RetryableError(errors.New("retry"))
		},
		func() (*http.Response, error) {
			if calls.Add(1) >= 2 {
				cancel()
			}
			return &http.Response{
				StatusCode: http.StatusServiceUnavailable,
				Body:       io.NopCloser(strings.NewReader("")),
			}, nil
		})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

// Retry-After: 0 should beat a large PollInterval so the second attempt runs quickly.
func TestRetryAPIRequest_retryAfterOverridesPollInterval(t *testing.T) {
	t.Parallel()
	var calls atomic.Int32
	var t1 time.Time
	err := cloud.RetryAPIRequest(context.Background(), time.Second, 500*time.Millisecond,
		func(err error, resp *http.Response) *retry.RetryError {
			if err != nil {
				return retry.NonRetryableError(err)
			}
			if resp.StatusCode == http.StatusTooManyRequests {
				return retry.RetryableError(errors.New("too many requests"))
			}
			return nil
		},
		func() (*http.Response, error) {
			n := calls.Add(1)
			if n == 1 {
				t1 = time.Now()
				return &http.Response{
					StatusCode: http.StatusTooManyRequests,
					Header:     http.Header{"Retry-After": []string{"0"}},
					Body:       io.NopCloser(strings.NewReader("")),
				}, nil
			}
			if elapsed := time.Since(t1); elapsed >= 200*time.Millisecond {
				t.Errorf("second call took %v, expected Retry-After 0 to skip 500ms poll", elapsed)
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader("")),
			}, nil
		})
	if err != nil {
		t.Fatal(err)
	}
}

func TestRetryAPIRequest_pollIntervalWhenNoRetryAfter(t *testing.T) {
	t.Parallel()
	var calls atomic.Int32
	var t1 time.Time
	const poll = 80 * time.Millisecond
	err := cloud.RetryAPIRequest(context.Background(), time.Second, poll,
		func(err error, resp *http.Response) *retry.RetryError {
			if err != nil {
				return retry.NonRetryableError(err)
			}
			if resp.StatusCode == http.StatusServiceUnavailable {
				return retry.RetryableError(errors.New("unavailable"))
			}
			return nil
		},
		func() (*http.Response, error) {
			n := calls.Add(1)
			if n == 1 {
				t1 = time.Now()
				return &http.Response{
					StatusCode: http.StatusServiceUnavailable,
					Body:       io.NopCloser(strings.NewReader("")),
				}, nil
			}
			if elapsed := time.Since(t1); elapsed < poll/2 {
				t.Errorf("second call after %v, expected at least ~%v without Retry-After", elapsed, poll/2)
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader("")),
			}, nil
		})
	if err != nil {
		t.Fatal(err)
	}
}

func TestRetryAPIRequest_httptestIntegration(t *testing.T) {
	t.Parallel()
	var attempts atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := attempts.Add(1)
		if n == 1 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	client := srv.Client()
	err := cloud.RetryAPIRequest(context.Background(), time.Second, 300*time.Millisecond,
		func(err error, resp *http.Response) *retry.RetryError {
			if err != nil {
				return retry.NonRetryableError(err)
			}
			if resp.StatusCode == http.StatusTooManyRequests {
				return retry.RetryableError(errors.New("too many requests"))
			}
			if resp.StatusCode != http.StatusOK {
				_, _ = io.Copy(io.Discard, resp.Body)
				_ = resp.Body.Close()
				return retry.NonRetryableError(fmt.Errorf("unexpected %d", resp.StatusCode))
			}
			_, _ = io.Copy(io.Discard, resp.Body)
			_ = resp.Body.Close()
			return nil
		},
		func() (*http.Response, error) {
			req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL, nil)
			if err != nil {
				return nil, err
			}
			return client.Do(req)
		})
	if err != nil {
		t.Fatal(err)
	}
	if attempts.Load() != 2 {
		t.Fatalf("expected 2 HTTP requests, got %d", attempts.Load())
	}
}

func TestGetRetryStrategy(t *testing.T) {
	t.Parallel()

	apiErr := errors.New("API failure")

	cases := []struct {
		name      string
		err       error
		resp      *http.Response
		wantNil   bool
		retryable bool
	}{
		{
			name:    "2xx success",
			resp:    &http.Response{StatusCode: http.StatusOK, Status: "200 OK"},
			wantNil: true,
		},
		{
			name:    "201 created",
			resp:    &http.Response{StatusCode: http.StatusCreated, Status: "201 Created"},
			wantNil: true,
		},
		{
			name:      "2xx with decode error not retryable",
			err:       apiErr,
			resp:      &http.Response{StatusCode: http.StatusOK, Status: "200 OK"},
			retryable: false,
		},
		{
			name:      "404 not retryable",
			resp:      &http.Response{StatusCode: http.StatusNotFound, Status: "404 Not Found"},
			retryable: false,
		},
		{
			name:      "404 with wrapped API error",
			err:       apiErr,
			resp:      &http.Response{StatusCode: http.StatusNotFound, Status: "404 Not Found"},
			retryable: false,
		},
		{
			name:      "429 retryable",
			resp:      &http.Response{StatusCode: http.StatusTooManyRequests, Status: "429 Too Many Requests"},
			retryable: true,
		},
		{
			name:      "429 retryable with error",
			err:       apiErr,
			resp:      &http.Response{StatusCode: http.StatusTooManyRequests, Status: "429 Too Many Requests"},
			retryable: true,
		},
		{
			name:      "500 retryable",
			resp:      &http.Response{StatusCode: http.StatusInternalServerError, Status: "500 Internal Server Error"},
			retryable: true,
		},
		{
			name:      "503 retryable",
			resp:      &http.Response{StatusCode: http.StatusServiceUnavailable, Status: "503 Service Unavailable"},
			retryable: true,
		},
		{
			name:      "599 retryable",
			resp:      &http.Response{StatusCode: 599, Status: "599 Server Error"},
			retryable: true,
		},
		{
			name:      "400 not retryable",
			resp:      &http.Response{StatusCode: http.StatusBadRequest, Status: "400 Bad Request"},
			retryable: false,
		},
		{
			name:      "nil response transport error not retryable",
			err:       apiErr,
			retryable: false,
		},
		{
			name:    "nil response nil error",
			wantNil: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := cloud.GetRetryStrategy(tc.err, tc.resp)
			switch {
			case tc.wantNil:
				if got != nil {
					t.Fatalf("expected nil, got %+v", got)
				}
			case got == nil:
				t.Fatal("expected non-nil decision")
			default:
				if got.Retryable != tc.retryable {
					t.Fatalf("Retryable=%v, want %v", got.Retryable, tc.retryable)
				}
				if tc.err != nil && !errors.Is(got.Err, tc.err) {
					t.Fatalf("Err=%v, want wraps %v", got.Err, tc.err)
				}
			}
		})
	}
}

func TestGetRetryStrategy_withRetryAPIRequest_429RetryAfter(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	var t0 time.Time

	err := cloud.RetryAPIRequest(context.Background(), time.Second, 400*time.Millisecond,
		cloud.GetRetryStrategy,
		func() (*http.Response, error) {
			n := calls.Add(1)
			if n == 1 {
				t0 = time.Now()
				return &http.Response{
					StatusCode: http.StatusTooManyRequests,
					Status:     "429 Too Many Requests",
					Header:     http.Header{"Retry-After": []string{"0"}},
					Body:       io.NopCloser(strings.NewReader("")),
				}, nil
			}
			if elapsed := time.Since(t0); elapsed >= 200*time.Millisecond {
				t.Errorf("second attempt after %v, wanted Retry-After 0 to beat 400ms poll", elapsed)
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Status:     "200 OK",
				Body:       io.NopCloser(strings.NewReader("")),
			}, nil
		})
	if err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 2 {
		t.Fatalf("want 2 attempts, got %d", calls.Load())
	}
}

func TestGetRetryStrategy_withRetryAPIRequest_404Stops(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	err := cloud.RetryAPIRequest(context.Background(), time.Second, time.Millisecond,
		cloud.GetRetryStrategy,
		func() (*http.Response, error) {
			calls.Add(1)
			return &http.Response{
				StatusCode: http.StatusNotFound,
				Status:     "404 Not Found",
				Body:       io.NopCloser(strings.NewReader("")),
			}, nil
		})
	if err == nil {
		t.Fatal("expected error")
	}
	if calls.Load() != 1 {
		t.Fatalf("want single attempt, got %d", calls.Load())
	}
}

func TestGetRetryStrategyAllowNotFound(t *testing.T) {
	t.Parallel()

	apiErr := errors.New("API failure")
	strategy := cloud.GetRetryStrategyAllowNotFound

	cases := []struct {
		name      string
		err       error
		resp      *http.Response
		wantNil   bool
		retryable bool
	}{
		{
			name:      "404 retryable",
			resp:      &http.Response{StatusCode: http.StatusNotFound, Status: "404 Not Found"},
			retryable: true,
		},
		{
			name:      "404 with API error retryable",
			err:       apiErr,
			resp:      &http.Response{StatusCode: http.StatusNotFound, Status: "404 Not Found"},
			retryable: true,
		},
		{
			name:    "200 success unchanged",
			resp:    &http.Response{StatusCode: http.StatusOK, Status: "200 OK"},
			wantNil: true,
		},
		{
			name:      "400 still not retryable",
			resp:      &http.Response{StatusCode: http.StatusBadRequest, Status: "400 Bad Request"},
			retryable: false,
		},
		{
			name:      "503 still retryable",
			resp:      &http.Response{StatusCode: http.StatusServiceUnavailable, Status: "503 Service Unavailable"},
			retryable: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := strategy(tc.err, tc.resp)
			switch {
			case tc.wantNil:
				if got != nil {
					t.Fatalf("expected nil, got %+v", got)
				}
			case got == nil:
				t.Fatal("expected non-nil decision")
			default:
				if got.Retryable != tc.retryable {
					t.Fatalf("Retryable=%v, want %v", got.Retryable, tc.retryable)
				}
				if tc.err != nil && !errors.Is(got.Err, tc.err) {
					t.Fatalf("Err=%v, want wraps %v", got.Err, tc.err)
				}
			}
		})
	}
}

func TestGetRetryStrategyAllowNotFound_withRetryAPIRequest_404ThenSuccess(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	err := cloud.RetryAPIRequest(context.Background(), time.Second, time.Millisecond,
		cloud.GetRetryStrategyAllowNotFound,
		func() (*http.Response, error) {
			n := calls.Add(1)
			if n == 1 {
				return &http.Response{
					StatusCode: http.StatusNotFound,
					Status:     "404 Not Found",
					Body:       io.NopCloser(strings.NewReader("")),
				}, nil
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Status:     "200 OK",
				Body:       io.NopCloser(strings.NewReader("")),
			}, nil
		})
	if err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 2 {
		t.Fatalf("want 2 attempts, got %d", calls.Load())
	}
}

func TestRetryAPIRequest_GetRetryStrategy_createPrecheck404IsNotFoundError(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	err := cloud.RetryAPIRequest(context.Background(), time.Second, time.Millisecond,
		cloud.GetRetryStrategy,
		func() (*http.Response, error) {
			calls.Add(1)
			return &http.Response{
				StatusCode: http.StatusNotFound,
				Status:     "404 Not Found",
				Body:       io.NopCloser(strings.NewReader("")),
			}, errors.New("404 Not Found")
		})
	if err == nil {
		t.Fatal("expected error from non-retryable 404")
	}
	if !common.IsNotFoundError(err) {
		t.Fatalf("expected IsNotFoundError for create-precheck slug-not-taken path, got %v", err)
	}
	if calls.Load() != 1 {
		t.Fatalf("want 1 attempt, got %d", calls.Load())
	}
}

func TestRetryAPIRequest_GetRetryStrategy_retries503Then404IsNotFoundError(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	err := cloud.RetryAPIRequest(context.Background(), time.Second, time.Millisecond,
		cloud.GetRetryStrategy,
		func() (*http.Response, error) {
			n := calls.Add(1)
			if n == 1 {
				return &http.Response{
					StatusCode: http.StatusBadGateway,
					Status:     "502 Bad Gateway",
					Body:       io.NopCloser(strings.NewReader("")),
				}, errors.New("502 Bad Gateway")
			}
			return &http.Response{
				StatusCode: http.StatusNotFound,
				Status:     "404 Not Found",
				Body:       io.NopCloser(strings.NewReader("")),
			}, errors.New("404 Not Found")
		})
	if err == nil {
		t.Fatal("expected error")
	}
	if !common.IsNotFoundError(err) {
		t.Fatalf("expected IsNotFoundError, got %v", err)
	}
	if calls.Load() != 2 {
		t.Fatalf("want 2 attempts, got %d", calls.Load())
	}
}
