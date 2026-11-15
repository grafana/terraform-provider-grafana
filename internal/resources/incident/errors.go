package incident

import (
	"errors"
	"fmt"
	"net/http"

	incident "github.com/grafana/incident-go"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// asAPIError unwraps the *incident.APIError the generated client returns for a
// non-2xx response.
func asAPIError(err error) (*incident.APIError, bool) {
	var apiErr *incident.APIError
	if errors.As(err, &apiErr) {
		return apiErr, true
	}
	return nil, false
}

// isNotFound reports whether the API answered 404 for the object the request
// named.
func isNotFound(err error) bool {
	apiErr, ok := asAPIError(err)
	return ok && apiErr.HTTPStatusCode == http.StatusNotFound
}

// incidentsInUse describes how many incidents use a severity, field or select
// option, from the incidents_count an in-use error carries, falling back to a
// vaguer phrase when the count is missing.
func incidentsInUse(err error) string {
	if apiErr, ok := asAPIError(err); ok {
		// JSON numbers decode into an interface{} as float64.
		if count, ok := apiErr.Context["incidents_count"].(float64); ok {
			if count == 1 {
				return "1 incident"
			}
			return fmt.Sprintf("%d incidents", int(count))
		}
	}
	return "one or more incidents"
}

// optionalString maps an optional string the API stores as "" when unset. An
// empty value is ambiguous on its own, so configured, the value from the plan or
// prior state, decides: "" stays "" when the practitioner set it, and becomes
// null when they left the attribute out. Mapping it to null unconditionally
// fails every apply of attr = "" with an inconsistent result.
func optionalString(value string, configured types.String) types.String {
	if value == "" && configured.IsNull() {
		return types.StringNull()
	}
	return types.StringValue(value)
}
