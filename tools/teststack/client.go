package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/grafana/grafana-com-public-clients/go/gcom"
)

// requestID returns a unique correlation ID for use as the gcom X-Request-ID
// header. The format mirrors the provider's ClientRequestID helper but stays
// independent of any internal package import cycle.
func requestID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		// rand.Read effectively never fails on supported platforms; on the
		// off chance it does, fall back to a timestamp-based ID.
		return fmt.Sprintf("teststack-%d", time.Now().UnixNano())
	}
	return "teststack-" + hex.EncodeToString(b[:])
}

// newGcomClient builds a gcom API client authenticated with the org-level CAP
// token from GRAFANA_CLOUD_ACCESS_POLICY_TOKEN.
func newGcomClient(cloudAPIURL, capToken string) (*gcom.APIClient, error) {
	cfg := gcom.NewConfiguration()
	u, err := url.Parse(cloudAPIURL)
	if err != nil {
		return nil, fmt.Errorf("parse cloud API URL %q: %w", cloudAPIURL, err)
	}
	cfg.Host = u.Host
	cfg.Scheme = u.Scheme
	cfg.HTTPClient = &http.Client{Timeout: 60 * time.Second}
	cfg.UserAgent = "terraform-provider-grafana-teststack/0.1"
	cfg.DefaultHeader["Authorization"] = "Bearer " + capToken
	return gcom.NewAPIClient(cfg), nil
}

// bearer wraps an http.Client to inject a Bearer token. Used for raw API calls
// (k6 install, fleet management, etc.) that don't have a generated client.
type bearerTransport struct {
	base  http.RoundTripper
	token string
}

func (b *bearerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if b.token != "" {
		req.Header.Set("Authorization", "Bearer "+b.token)
	}
	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", "terraform-provider-grafana-teststack/0.1")
	}
	return b.base.RoundTrip(req)
}

func bearerClient(token string) *http.Client {
	return &http.Client{
		Transport: &bearerTransport{base: http.DefaultTransport, token: token},
		Timeout:   60 * time.Second,
	}
}

// pollUntil invokes check until it returns done=true or ctx is cancelled.
// Returns ctx.Err() on timeout, or the last error from check.
func pollUntil(ctx context.Context, interval time.Duration, check func(context.Context) (done bool, err error)) error {
	for {
		done, err := check(ctx)
		if done {
			return nil
		}
		if err != nil {
			// Allow non-final errors to be surfaced via a final timeout error.
			// The caller logs them via the check function if desired.
			_ = err
		}
		select {
		case <-ctx.Done():
			if err != nil {
				return fmt.Errorf("%w (last error: %v)", ctx.Err(), err)
			}
			return ctx.Err()
		case <-time.After(interval):
		}
	}
}
