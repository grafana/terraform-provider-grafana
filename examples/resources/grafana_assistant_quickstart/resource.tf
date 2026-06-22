resource "grafana_assistant_quickstart" "example" {
  scope  = "tenant"
  title  = "SLO health"
  prompt = "How healthy are my SLOs right now?"
}
