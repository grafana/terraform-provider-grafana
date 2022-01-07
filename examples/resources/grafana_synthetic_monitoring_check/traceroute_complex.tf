data "grafana_synthetic_monitoring_probes" "main" {}

resource "grafana_synthetic_monitoring_check" "traceroute" {
  job       = "Traceroute complex"
  target    = "grafana.net"
  enabled   = false
  frequency = 120000
  timeout   = 30000
  probes = [
    data.grafana_synthetic_monitoring_probes.main.probes.Frankfurt,
    data.grafana_synthetic_monitoring_probes.main.probes.London,
  ]
  labels = {
    foo = "baz"
  }
  settings {
    traceroute {
      max_hops         = 25
      max_unknown_hops = 10
      ptr_lookup       = false
    }
  }
}
