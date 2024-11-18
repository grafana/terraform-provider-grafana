package provider

import (
	"net/http"

	"github.com/go-openapi/runtime"
	httptransport "github.com/go-openapi/runtime/client"
	"github.com/go-openapi/runtime/middleware"
)

type roundTripper struct {
	originalTransport http.RoundTripper
}

func newRoundTripper(originalClientTransport runtime.ClientTransport) http.RoundTripper {
	return &roundTripper{originalClientTransport.(*httptransport.Runtime).Transport}
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
