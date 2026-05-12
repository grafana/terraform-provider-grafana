package cloud

import "net/http"

// shouldRetryServiceAccountOperation reports whether a failed Grafana Cloud instance
// API POST (service account or token creation) should be retried: transport errors, rate limits,
// transient server errors, and 400 responses that may clear after stack provisioning or races.
func shouldRetryServiceAccountOperation(httpResp *http.Response, err error) bool {
	if err != nil && httpResp == nil {
		return true
	}
	if httpResp == nil {
		return false
	}
	switch code := httpResp.StatusCode; {
	case code == http.StatusTooManyRequests:
		return true
	case code >= http.StatusInternalServerError:
		return true
	case code == http.StatusBadRequest:
		return true
	default:
		return false
	}
}

// is5xxOrNetworkError reports whether a failed POST had no HTTP
// response with a non-nil error (typically transport) or a 5xx status.
func is5xxOrNetworkError(httpResp *http.Response, err error) bool {
	if err != nil && httpResp == nil {
		return true
	}
	if httpResp != nil && httpResp.StatusCode >= http.StatusInternalServerError {
		return true
	}
	return false
}

// shouldAdoptResource is true when an earlier attempt failed with
// a 5xx or network error and the current failure is HTTP 400 — a pattern that can mean the create
// succeeded server-side but the client did not receive the success response.
func shouldAdoptResource(sawPrior5xxOrNetwork bool, httpResp *http.Response) bool {
	return sawPrior5xxOrNetwork && httpResp != nil && httpResp.StatusCode == http.StatusBadRequest
}
