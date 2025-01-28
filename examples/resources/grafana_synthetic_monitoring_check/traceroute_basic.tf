data "grafana_synthetic_monitoring_probes" "main" {}

resource "grafana_synthetic_monitoring_check" "traceroute" {
  job       = "Traceroute defaults"
  target    = "grafana.com"
  enabled   = false
  frequency = 120000
  timeout   = 30000
  probes = [
    data.grafana_synthetic_monitoring_probes.main.probes.Ohio,
  ]
  labels = {
    foo = "bar"
  }
  settings {
    traceroute {}
  }
}
