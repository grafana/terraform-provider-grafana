resource "grafana_synthetic_monitoring_check" "multihttp" {
  job                 = "multihttp basic"
  target              = "https://www.grafana-dev.com"
  enabled             = false
  select_probes_count = 1
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
