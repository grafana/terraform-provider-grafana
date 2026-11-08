resource "grafana_assistant_rule" "example" {
  name         = "Prefer RED metrics"
  rule_content = "When summarizing service health, prefer RED metrics."
  scope        = "tenant"
  priority     = 10
  applications = ["assistant"]
}
