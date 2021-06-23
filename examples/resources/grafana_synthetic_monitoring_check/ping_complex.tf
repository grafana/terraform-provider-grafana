data "grafana_synthetic_monitoring_probes" "main" {}

resource "grafana_synthetic_monitoring_check" "ping" {
  job     = "Ping Updated"
  target  = "grafana.net"
  enabled = false
  probes = [
    data.grafana_synthetic_monitoring_probes.main.probes.Chicago,
    data.grafana_synthetic_monitoring_probes.main.probes.LosAngeles,
  ]
  labels = {
    foo = "baz"
  }
  settings {
    ping {
      ip_version    = "Any"
      payload_size  = 20
      dont_fragment = true
    }
  }
}
