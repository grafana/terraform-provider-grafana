package cloud

import "net/http"

// postInstanceStackServiceAccountWriteShouldRetry reports whether a failed Grafana Cloud instance
// API POST (service account or token creation) should be retried: rate limits, transient server
// errors, and 400 responses that may clear after stack provisioning or due to races on the instance.
func postInstanceStackServiceAccountWriteShouldRetry(httpResp *http.Response) bool {
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
