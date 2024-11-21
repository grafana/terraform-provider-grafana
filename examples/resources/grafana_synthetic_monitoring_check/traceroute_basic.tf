resource "grafana_synthetic_monitoring_check" "traceroute" {
  job                 = "Traceroute defaults"
  target              = "grafana.com"
  enabled             = false
  frequency           = 120000
  timeout             = 30000
  select_probes_count = 1
  labels = {
    foo = "bar"
  }
  settings {
    traceroute {}
  }
}
