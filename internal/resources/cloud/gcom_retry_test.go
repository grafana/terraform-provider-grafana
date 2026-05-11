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
