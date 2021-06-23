resource "grafana_synthetic_monitoring_probe" "main" {
  name      = "Mauna Loa"
  latitude  = 19.47948
  longitude = -155.60282
  region    = "AMER"
  labels = {
    type = "volcano"
  }
}
