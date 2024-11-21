resource "grafana_synthetic_monitoring_check" "dns" {
  job                 = "DNS Defaults"
  target              = "grafana.com"
  enabled             = false
  select_probes_count = 1
  labels = {
    foo = "bar"
  }
  settings {
    dns {}
  }
}
