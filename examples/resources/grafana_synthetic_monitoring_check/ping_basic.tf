resource "grafana_synthetic_monitoring_check" "ping" {
  job                 = "Ping Defaults"
  target              = "grafana.com"
  enabled             = false
  select_probes_count = 1
  labels = {
    foo = "bar"
  }
  settings {
    ping {}
  }
}
