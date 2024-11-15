package provider

import (
	"net/http"

	"github.com/go-openapi/runtime/middleware"
)

type roundTripper struct {
	originalTransport http.RoundTripper
}

func newRoundTripper(originalTransport http.RoundTripper) http.RoundTripper {
	return &roundTripper{originalTransport: originalTransport}
}

func (c *roundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := c.originalTransport.RoundTrip(req)
	if err != nil {
		return nil, err
	}

	mid := middleware.NegotiateContentType(req, []string{"application/json", "application/text"}, "application/json")
	resp.Header.Set("Content-Type", mid)
	return resp, nil
}
