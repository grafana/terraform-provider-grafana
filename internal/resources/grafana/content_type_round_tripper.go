package grafana

import (
	"net/http"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/runtime/middleware"
)

func ContentTypeNegotiator(tripper http.RoundTripper) func(operation *runtime.ClientOperation) {
	return func(operation *runtime.ClientOperation) {
		operation.Client = &http.Client{Transport: newContentTypeRoundTripperTest(tripper)}
	}
}

// contentTypeRoundTripperTest identifies unexpected "text/pain" responses when "application/json" is expected.
// It could happen when something fails related to an "implementation bug".
type contentTypeRoundTripperTest struct {
	originalRoundTripper http.RoundTripper
}

func newContentTypeRoundTripperTest(originalRoundTripper http.RoundTripper) http.RoundTripper {
	return &contentTypeRoundTripperTest{originalRoundTripper: originalRoundTripper}
}

func (r *contentTypeRoundTripperTest) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := r.originalRoundTripper.RoundTrip(req)
	if err != nil {
		return nil, err
	}

	mid := middleware.NegotiateContentType(req, []string{"application/json", "application/text"}, "application/text")
	resp.Header.Set("Content-Type", mid)

	return resp, nil
}
