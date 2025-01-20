data "grafana_synthetic_monitoring_probes" "main" {}

resource "grafana_synthetic_monitoring_check" "tcp" {
  job     = "TCP Defaults"
  target  = "grafana.com:80"
  enabled = false
  probes = [
    data.grafana_synthetic_monitoring_probes.main.probes.Ohio,
  ]
  labels = {
    foo = "bar"
  }
  settings {
    tcp {}
  }
}
