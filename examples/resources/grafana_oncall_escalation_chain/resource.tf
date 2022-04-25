resource "grafana_oncall_escalation_chain" "default" {
  provider = grafana.oncall
  name     = "default"
}
