data "grafana_synthetic_monitoring_probes" "main" {}

resource "grafana_synthetic_monitoring_check" "dns" {
  job     = "DNS Defaults"
  target  = "grafana.com"
  enabled = false
  probes = [
    data.grafana_synthetic_monitoring_probes.main.probes.Ohio,
  ]
  labels = {
    foo = "bar"
  }
  settings {
    dns {}
  }
}
