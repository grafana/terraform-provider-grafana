# Requires an access policy token with the adaptive-traces-policies:read and
# adaptive-traces-policies:write scopes, set as adaptive_traces_auth.
#
# Creating the first policy on a stack starts Adaptive Traces filtering, and
# deleting the last one stops it.

# The default policies that Adaptive Traces suggests as a starting point.
resource "grafana_adaptive_traces_policy" "errors" {
  name = "Keep all traces with errors"
  type = "status_code"
  body = jsonencode({
    status_codes = ["ERROR"]
  })
}

resource "grafana_adaptive_traces_policy" "diversity" {
  name = "Sample unique traces"
  type = "diversity"
  body = jsonencode({})
}

resource "grafana_adaptive_traces_policy" "latency" {
  name = "Sample slow traces"
  type = "latency"
  body = jsonencode({
    threshold_ms = 8000
  })
}

resource "grafana_adaptive_traces_policy" "volumetric" {
  name = "Sample 5 percent of traces by volume"
  type = "volumetric"
  body = jsonencode({
    sampling_percentage = 5
  })
}

# The API only accepts a drop policy while an active policy of another type
# exists, so make it depend on one to create it last and destroy it first.
resource "grafana_adaptive_traces_policy" "drop_health_checks" {
  name = "Drop health check traces"
  type = "drop"
  body = jsonencode({
    drop_sub_policy = [
      {
        name = "health-checks"
        type = "string_attribute"
        string_attribute = {
          key    = "url.path"
          values = ["/health", "/metrics"]
        }
      }
    ]
  })

  depends_on = [grafana_adaptive_traces_policy.errors]
}
