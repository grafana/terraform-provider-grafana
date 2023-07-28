data "grafana_synthetic_monitoring_probes" "main" {}

resource "grafana_synthetic_monitoring_check" "multihttp" {
  job     = "multihttp basic"
  target  = "https://www.grafana-dev.com"
  enabled = false
  probes = [
    data.grafana_synthetic_monitoring_probes.main.probes.Amsterdam,
  ]
  labels = {
    foo = "bar"
  }
  settings {
    multihttp {
      entries {
        request {
          method = "GET"
          url    = "https://www.grafana-dev.com"
        }
      }
    }
  }
}
