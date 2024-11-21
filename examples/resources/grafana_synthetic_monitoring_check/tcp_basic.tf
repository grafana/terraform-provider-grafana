resource "grafana_synthetic_monitoring_check" "tcp" {
  job                 = "TCP Defaults"
  target              = "grafana.com:80"
  enabled             = false
  select_probes_count = 1
  labels = {
    foo = "bar"
  }
  settings {
    tcp {}
  }
}
