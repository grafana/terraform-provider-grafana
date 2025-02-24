data "grafana_synthetic_monitoring_probes" "main" {}

resource "grafana_synthetic_monitoring_check" "http" {
  job     = "HTTP Defaults"
  target  = "https://grafana.com"
  enabled = false
  probes = [
    data.grafana_synthetic_monitoring_probes.main.probes.Ohio,
  ]
  labels = {
    foo = "bar"
  }
  settings {
    http {}
  }
}
