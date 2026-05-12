package cloud

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/grafana/grafana-com-public-clients/go/gcom"
)

type stub409BodyErr struct {
	body []byte
}

func (e stub409BodyErr) Error() string { return "stub" }
func (e stub409BodyErr) Body() []byte  { return e.body }

func TestGraceSleepForTransient409Conflict(t *testing.T) {
	t.Parallel()

	t.Run("no matching error type", func(t *testing.T) {
		t.Parallel()
		_, ok := graceSleepForTransient409Conflict(errors.New("plain"), "myslug")
		if ok {
			t.Fatal("expected false")
		}
	})

	t.Run("body missing keywords", func(t *testing.T) {
		t.Parallel()
		_, ok := graceSleepForTransient409Conflict(stub409BodyErr{body: []byte("something else")}, "myslug")
		if ok {
			t.Fatal("expected false")
		}
	})

	t.Run("deleted recently default 35s", func(t *testing.T) {
		t.Parallel()
		wait, ok := graceSleepForTransient409Conflict(stub409BodyErr{
			body: []byte(`deleted recently please wait`),
		}, "myslug")
		if !ok || wait != 35*time.Second {
			t.Fatalf("got ok=%v wait=%v", ok, wait)
		}
	})

	t.Run("slug already exists phrase default 35s", func(t *testing.T) {
		t.Parallel()
		wait, ok := graceSleepForTransient409Conflict(stub409BodyErr{
			body: []byte(`Grafana stack with the same slug already exists`),
		}, "myslug")
		if !ok || wait != 35*time.Second {
			t.Fatalf("got ok=%v wait=%v", ok, wait)
		}
	})

	t.Run("grace period from body", func(t *testing.T) {
		t.Parallel()
		wait, ok := graceSleepForTransient409Conflict(stub409BodyErr{
			body: []byte(`deleted recently, wait for 28s before retry`),
		}, "myslug")
		if !ok || wait != 33*time.Second {
			t.Fatalf("got ok=%v wait=%v want 33s", ok, wait)
		}
	})

	t.Run("invalid seconds keeps default", func(t *testing.T) {
		t.Parallel()
		wait, ok := graceSleepForTransient409Conflict(stub409BodyErr{
			body: []byte(`deleted recently wait for Xs`),
		}, "myslug")
		if !ok || wait != 35*time.Second {
			t.Fatalf("got ok=%v wait=%v", ok, wait)
		}
	})
}

func TestDecideCreateStackOuterRetry(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	slug := "test-slug"
	apiErr := errors.New("wrapped api failure")

	t.Run("no exec error returns apiErr", func(t *testing.T) {
		t.Parallel()
		dec := DecideCreateStackOuterRetry(ctx, slug, apiErr, nil, nil, func(context.Context, string) (*gcom.FormattedApiInstance, error) {
			t.Fatal("getInstance should not be called")
			return nil, nil
		})
		if dec.StopWithErr != apiErr || dec.AdoptedInstance != nil || dec.SleepBeforeContinue != 0 {
			t.Fatalf("unexpected decision: %+v", dec)
		}
	})

	t.Run("non-conflict adopts existing stack", func(t *testing.T) {
		t.Parallel()
		inst := &gcom.FormattedApiInstance{Id: 42}
		dec := DecideCreateStackOuterRetry(ctx, slug, apiErr, errors.New("504"), &http.Response{StatusCode: http.StatusGatewayTimeout},
			func(context.Context, string) (*gcom.FormattedApiInstance, error) {
				return inst, nil
			})
		if dec.AdoptedInstance != inst || dec.StopWithErr != nil || dec.SleepBeforeContinue != 0 {
			t.Fatalf("unexpected decision: %+v", dec)
		}
	})

	t.Run("non-conflict getInstance fails sleeps 10s", func(t *testing.T) {
		t.Parallel()
		dec := DecideCreateStackOuterRetry(ctx, slug, apiErr, errors.New("504"), &http.Response{StatusCode: http.StatusGatewayTimeout},
			func(context.Context, string) (*gcom.FormattedApiInstance, error) {
				return nil, errors.New("not found")
			})
		if dec.SleepBeforeContinue != 10*time.Second || dec.StopWithErr != nil || dec.AdoptedInstance != nil {
			t.Fatalf("unexpected decision: %+v", dec)
		}
	})

	t.Run("409 transient grace sleep", func(t *testing.T) {
		t.Parallel()
		dec := DecideCreateStackOuterRetry(ctx, slug, apiErr,
			stub409BodyErr{body: []byte(`deleted recently wait for 10s`)},
			&http.Response{StatusCode: http.StatusConflict},
			func(context.Context, string) (*gcom.FormattedApiInstance, error) {
				t.Fatal("getInstance should not be called")
				return nil, nil
			})
		if dec.SleepBeforeContinue != 15*time.Second || dec.StopWithErr != nil || dec.AdoptedInstance != nil {
			t.Fatalf("unexpected decision: %+v", dec)
		}
	})

	t.Run("409 existing active stack", func(t *testing.T) {
		t.Parallel()
		existing := &gcom.FormattedApiInstance{Id: 7, Status: "active"}
		dec := DecideCreateStackOuterRetry(ctx, slug, apiErr,
			errors.New("conflict"),
			&http.Response{StatusCode: http.StatusConflict},
			func(context.Context, string) (*gcom.FormattedApiInstance, error) {
				return existing, nil
			})
		if dec.StopWithErr == nil || dec.AdoptedInstance != nil || dec.SleepBeforeContinue != 0 {
			t.Fatalf("unexpected decision: %+v", dec)
		}
		if !strings.Contains(dec.StopWithErr.Error(), slug) || !strings.Contains(dec.StopWithErr.Error(), "already used") {
			t.Fatalf("unexpected error: %v", dec.StopWithErr)
		}
	})

	t.Run("409 falls through to lastExecErr", func(t *testing.T) {
		t.Parallel()
		conflictErr := errors.New("hard conflict")
		dec := DecideCreateStackOuterRetry(ctx, slug, apiErr,
			conflictErr,
			&http.Response{StatusCode: http.StatusConflict},
			func(context.Context, string) (*gcom.FormattedApiInstance, error) {
				return nil, errors.New("lookup failed")
			})
		if dec.StopWithErr != conflictErr || dec.AdoptedInstance != nil || dec.SleepBeforeContinue != 0 {
			t.Fatalf("unexpected decision: %+v", dec)
		}
	})

	t.Run("409 deleted stack lookup returns deleted status", func(t *testing.T) {
		t.Parallel()
		deleted := &gcom.FormattedApiInstance{Id: 1, Status: "deleted"}
		conflictErr := errors.New("still conflict")
		dec := DecideCreateStackOuterRetry(ctx, slug, apiErr,
			conflictErr,
			&http.Response{StatusCode: http.StatusConflict},
			func(context.Context, string) (*gcom.FormattedApiInstance, error) {
				return deleted, nil
			})
		if dec.StopWithErr != conflictErr || dec.AdoptedInstance != nil || dec.SleepBeforeContinue != 0 {
			t.Fatalf("unexpected decision: %+v", dec)
		}
	})
}

func TestNeedsAnotherCreateStackHTTPAttempt(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		resp *http.Response
		want bool
	}{
		{name: "nil response", resp: nil, want: false},
		{name: "429 Too Many Requests", resp: &http.Response{StatusCode: http.StatusTooManyRequests}, want: true},
		{name: "500 Internal Server Error", resp: &http.Response{StatusCode: http.StatusInternalServerError}, want: true},
		{name: "502 Bad Gateway", resp: &http.Response{StatusCode: http.StatusBadGateway}, want: true},
		{name: "503 Service Unavailable", resp: &http.Response{StatusCode: http.StatusServiceUnavailable}, want: true},
		{name: "599", resp: &http.Response{StatusCode: 599}, want: true},
		{name: "409 Conflict", resp: &http.Response{StatusCode: http.StatusConflict}, want: false},
		{name: "404 Not Found", resp: &http.Response{StatusCode: http.StatusNotFound}, want: false},
		{name: "400 Bad Request", resp: &http.Response{StatusCode: http.StatusBadRequest}, want: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := needsAnotherCreateStackHTTPAttempt(tc.resp); got != tc.want {
				t.Fatalf("needsAnotherCreateStackHTTPAttempt()=%v, want %v", got, tc.want)
			}
		})
	}
}
