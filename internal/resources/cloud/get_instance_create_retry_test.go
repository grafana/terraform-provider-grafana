package cloud

import (
	"net/http"
	"testing"
	"time"
)

func TestStackCreateLaterDeadline(t *testing.T) {
	t.Parallel()
	start := time.Unix(100, 0)
	a := start.Add(1 * time.Minute)
	b := start.Add(2 * time.Minute)
	if got := stackCreateLaterDeadline(a, b); !got.Equal(b) {
		t.Fatalf("expected later of a,b to be b, got %v", got)
	}
	if got := stackCreateLaterDeadline(b, a); !got.Equal(b) {
		t.Fatalf("expected later of b,a to be b, got %v", got)
	}
}

func TestStackCreateGetInstanceRetryDecision(t *testing.T) {
	t.Parallel()

	start := time.Unix(1000, 0)
	existsBudgetEnd := start.Add(createStackGetInstanceExistsBudget)
	transientBudgetEnd := start.Add(createStackGetInstanceTransientBudget)
	initialDeadline := start.Add(createStackGetInstanceExistsBudget)

	cases := []struct {
		name              string
		retryNotFound     bool
		now               time.Time
		effectiveDeadline time.Time
		resp              *http.Response
		wantRetry         bool
		wantDeadline      time.Time
	}{
		{
			name:              "nil response",
			retryNotFound:     true,
			now:               start.Add(10 * time.Second),
			effectiveDeadline: initialDeadline,
			resp:              nil,
			wantRetry:         false,
			wantDeadline:      initialDeadline,
		},
		{
			name:              "404 retry until exists budget",
			retryNotFound:     true,
			now:               start.Add(30 * time.Second),
			effectiveDeadline: initialDeadline,
			resp:              &http.Response{StatusCode: http.StatusNotFound},
			wantRetry:         true,
			wantDeadline:      initialDeadline,
		},
		{
			name:              "404 no retry after exists budget",
			retryNotFound:     true,
			now:               existsBudgetEnd.Add(time.Second),
			effectiveDeadline: initialDeadline,
			resp:              &http.Response{StatusCode: http.StatusNotFound},
			wantRetry:         false,
			wantDeadline:      initialDeadline,
		},
		{
			name:              "404 never when retryNotFound false",
			retryNotFound:     false,
			now:               start.Add(10 * time.Second),
			effectiveDeadline: initialDeadline,
			resp:              &http.Response{StatusCode: http.StatusNotFound},
			wantRetry:         false,
			wantDeadline:      initialDeadline,
		},
		{
			name:              "429 extends deadline and retries",
			retryNotFound:     false,
			now:               start.Add(10 * time.Second),
			effectiveDeadline: initialDeadline,
			resp:              &http.Response{StatusCode: http.StatusTooManyRequests},
			wantRetry:         true,
			wantDeadline:      transientBudgetEnd,
		},
		{
			name:              "429 past transient budget end",
			retryNotFound:     false,
			now:               transientBudgetEnd.Add(time.Second),
			effectiveDeadline: transientBudgetEnd,
			resp:              &http.Response{StatusCode: http.StatusTooManyRequests},
			wantRetry:         false,
			wantDeadline:      transientBudgetEnd,
		},
		{
			name:              "503 extends deadline like 429",
			retryNotFound:     false,
			now:               start.Add(5 * time.Second),
			effectiveDeadline: initialDeadline,
			resp:              &http.Response{StatusCode: http.StatusBadGateway},
			wantRetry:         true,
			wantDeadline:      transientBudgetEnd,
		},
		{
			name:              "599 extends deadline",
			retryNotFound:     false,
			now:               start.Add(5 * time.Second),
			effectiveDeadline: initialDeadline,
			resp:              &http.Response{StatusCode: 599},
			wantRetry:         true,
			wantDeadline:      transientBudgetEnd,
		},
		{
			name:              "400 no retry",
			retryNotFound:     true,
			now:               start.Add(5 * time.Second),
			effectiveDeadline: initialDeadline,
			resp:              &http.Response{StatusCode: http.StatusBadRequest},
			wantRetry:         false,
			wantDeadline:      initialDeadline,
		},
		{
			name:              "429 keeps later deadline if already extended",
			retryNotFound:     false,
			now:               start.Add(30 * time.Second),
			effectiveDeadline: transientBudgetEnd,
			resp:              &http.Response{StatusCode: http.StatusTooManyRequests},
			wantRetry:         true,
			wantDeadline:      transientBudgetEnd,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			gotRetry, gotDeadline := stackCreateGetInstanceRetryDecision(tc.retryNotFound, start, tc.now, tc.effectiveDeadline, tc.resp)
			if gotRetry != tc.wantRetry || !gotDeadline.Equal(tc.wantDeadline) {
				t.Fatalf("retry=%v deadline=%v want retry=%v deadline=%v", gotRetry, gotDeadline, tc.wantRetry, tc.wantDeadline)
			}
		})
	}
}
