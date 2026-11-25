# Requires an access policy token with the adaptive-traces-policies:read and
# adaptive-traces-policies:write scopes, set as adaptive_traces_auth.
#
# Creating the first policy on a stack starts Adaptive Traces filtering, and
# deleting the last one stops it.

resource "grafana_adaptive_traces_policy" "errors" {
  name = "Keep all traces with errors"
  type = "status_code"
  body = jsonencode({
    status_codes = ["ERROR"]
  })
}
