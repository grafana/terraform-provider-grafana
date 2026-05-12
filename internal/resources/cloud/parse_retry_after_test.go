package cloud

import (
	"net/http"
	"testing"
	"time"
)

func TestParseRetryAfter_deltaSeconds(t *testing.T) {
	t.Parallel()
	resp := &http.Response{
		Header: http.Header{"Retry-After": []string{" 42 "}},
	}
	d, ok := parseRetryAfter(resp)
	if !ok || d != 42*time.Second {
		t.Fatalf("got (%v, %v)", d, ok)
	}
}

func TestParseRetryAfter_HTTPDate(t *testing.T) {
	t.Parallel()
	when := time.Now().Add(73 * time.Minute).UTC()
	resp := &http.Response{
		Header: http.Header{"Retry-After": []string{when.Format(http.TimeFormat)}},
	}
	d, ok := parseRetryAfter(resp)
	if !ok {
		t.Fatal("expected parsed Retry-After HTTP-date")
	}
	min := 72 * time.Minute
	max := 74 * time.Minute
	if d < min || d > max {
		t.Fatalf("duration %v outside [%v, %v]", d, min, max)
	}
}

func TestParseRetryAfter_absentOrInvalid(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		resp *http.Response
	}{
		{"nil response", nil},
		{"no header", &http.Response{Header: http.Header{}}},
		{"invalid", &http.Response{Header: http.Header{"Retry-After": []string{"not-a-number-or-date"}}}},
		{"negative delta", &http.Response{Header: http.Header{"Retry-After": []string{"-1"}}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if _, ok := parseRetryAfter(tc.resp); ok {
				t.Fatal("expected false")
			}
		})
	}
}

func TestParseRetryAfter_pastHTTPDate(t *testing.T) {
	t.Parallel()
	past := time.Now().Add(-5 * time.Minute).UTC()
	resp := &http.Response{
		Header: http.Header{"Retry-After": []string{past.Format(http.TimeFormat)}},
	}
	d, ok := parseRetryAfter(resp)
	if !ok || d != 0 {
		t.Fatalf("expected immediate retry (0), got (%v, %v)", d, ok)
	}
}
