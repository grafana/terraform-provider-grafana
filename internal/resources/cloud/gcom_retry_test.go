package cloud

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestUnitRetryGCOM_TreatNotFoundAsSuccess(t *testing.T) {
	ctx := context.Background()
	errTreat := RetryGCOM(ctx, GCOMRetryConfig{TreatNotFoundAsSuccess: true}, func() (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusNotFound,
			Body:       io.NopCloser(strings.NewReader("{}")),
		}, errors.New("not found")
	})
	if errTreat != nil {
		t.Fatalf("expected nil for DELETE-style 404, got %v", errTreat)
	}
	errUntreated := RetryGCOM(ctx, GCOMRetryConfig{}, func() (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusNotFound,
			Body:       io.NopCloser(strings.NewReader("{}")),
		}, errors.New("not found")
	})
	if errUntreated == nil {
		t.Fatal("expected non-nil error when TreatNotFoundAsSuccess is false")
	}
}

func TestUnitParseRetryAfter_DeltaSeconds(t *testing.T) {
	t.Parallel()
	fixed := time.Date(2026, time.March, 1, 12, 0, 0, 0, time.UTC)
	d, ok := parseRetryAfter("120", fixed)
	if !ok || d != 120*time.Second {
		t.Fatalf("want 120s, ok=true; got %v, ok=%v", d, ok)
	}
	d, ok = parseRetryAfter("0", fixed)
	if !ok || d != 0 {
		t.Fatalf("want 0, ok=true; got %v, ok=%v", d, ok)
	}
}

func TestUnitParseRetryAfter_HTTPDate(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, time.June, 1, 12, 0, 0, 0, time.UTC)
	when := now.Add(90 * time.Second)
	header := when.UTC().Format("Mon, 02 Jan 2006 15:04:05 GMT")
	d, ok := parseRetryAfter(header, now)
	if !ok {
		t.Fatal("want ok=true")
	}
	if d < 89*time.Second || d > 91*time.Second {
		t.Fatalf("want ~90s, got %v", d)
	}
	if d2, ok2 := parseRetryAfter(header, now.Add(300*time.Second)); !ok2 || d2 != 0 {
		t.Fatalf("past date should yield 0 wait: got %v ok=%v", d2, ok2)
	}
}

func TestUnitParseRetryAfter_Invalid(t *testing.T) {
	t.Parallel()
	now := time.Now()
	for _, h := range []string{"", "   ", "-1", "nan", "Thu, bogus"} {
		if _, ok := parseRetryAfter(h, now); ok {
			t.Fatalf("header %q should be invalid", h)
		}
	}
}

func TestUnitRetryGCOM_RetriesOn429AndHonoursRetryAfter(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	attempts := 0
	start := time.Now()
	err := RetryGCOM(ctx, GCOMRetryConfig{}, func() (*http.Response, error) {
		attempts++
		if attempts == 1 {
			return &http.Response{
				StatusCode: http.StatusTooManyRequests,
				Header:     http.Header{"Retry-After": []string{"1"}},
				Body:       io.NopCloser(strings.NewReader("{}")),
			}, errors.New("429 Too Many Requests")
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader("{}")),
		}, nil
	})
	if err != nil {
		t.Fatalf("expected success after retry, got %v", err)
	}
	if attempts != 2 {
		t.Fatalf("expected 2 attempts, got %d", attempts)
	}
	if elapsed := time.Since(start); elapsed < time.Second {
		t.Fatalf("expected to wait at least 1s for Retry-After, waited %v", elapsed)
	}
}

func TestUnitRetryGCOM_OnlyRetryRateLimited(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	// 5xx must not be retried for non-idempotent operations.
	attempts := 0
	err := RetryGCOM(ctx, GCOMRetryConfig{OnlyRetryRateLimited: true}, func() (*http.Response, error) {
		attempts++
		return &http.Response{
			StatusCode: http.StatusBadGateway,
			Body:       io.NopCloser(strings.NewReader("{}")),
		}, errors.New("502 Bad Gateway")
	})
	if err == nil {
		t.Fatal("expected error for 502 response")
	}
	if attempts != 1 {
		t.Fatalf("expected exactly 1 attempt for 5xx with OnlyRetryRateLimited, got %d", attempts)
	}

	// 429 is rejected before processing, so it stays retryable.
	attempts = 0
	err = RetryGCOM(ctx, GCOMRetryConfig{OnlyRetryRateLimited: true}, func() (*http.Response, error) {
		attempts++
		if attempts == 1 {
			return &http.Response{
				StatusCode: http.StatusTooManyRequests,
				Header:     http.Header{"Retry-After": []string{"0"}},
				Body:       io.NopCloser(strings.NewReader("{}")),
			}, errors.New("429 Too Many Requests")
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader("{}")),
		}, nil
	})
	if err != nil {
		t.Fatalf("expected success after 429 retry, got %v", err)
	}
	if attempts != 2 {
		t.Fatalf("expected 2 attempts, got %d", attempts)
	}
}

func TestUnitRetryGCOM_GivesUpWhenRetryAfterExceedsBudget(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	attempts := 0
	start := time.Now()
	err := RetryGCOM(ctx, GCOMRetryConfig{Timeout: 2 * time.Second}, func() (*http.Response, error) {
		attempts++
		return &http.Response{
			StatusCode: http.StatusTooManyRequests,
			Header:     http.Header{"Retry-After": []string{"3600"}},
			Body:       io.NopCloser(strings.NewReader("{}")),
		}, errors.New("429 Too Many Requests")
	})
	if err == nil {
		t.Fatal("expected error when Retry-After exceeds retry budget")
	}
	if !strings.Contains(err.Error(), "exceeds the remaining retry budget") {
		t.Fatalf("expected budget-exceeded error, got %v", err)
	}
	if attempts != 1 {
		t.Fatalf("expected exactly 1 attempt, got %d", attempts)
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Fatalf("expected immediate give-up without sleeping, took %v", elapsed)
	}
}

func TestUnitRetryGCOM_NonRetryableClientError(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	attempts := 0
	err := RetryGCOM(ctx, GCOMRetryConfig{}, func() (*http.Response, error) {
		attempts++
		return &http.Response{
			StatusCode: http.StatusBadRequest,
			Body:       io.NopCloser(strings.NewReader("{}")),
		}, errors.New("400 Bad Request")
	})
	if err == nil {
		t.Fatal("expected error for 400 response")
	}
	if attempts != 1 {
		t.Fatalf("expected exactly 1 attempt for non-retryable error, got %d", attempts)
	}
}
