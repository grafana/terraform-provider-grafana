resource "grafana_synthetic_monitoring_check" "http" {
  job                 = "HTTP Defaults"
  target              = "https://grafana.com"
  enabled             = false
  select_probes_count = 1
  labels = {
    foo = "bar"
  }
  settings {
    http {}
  }
}
