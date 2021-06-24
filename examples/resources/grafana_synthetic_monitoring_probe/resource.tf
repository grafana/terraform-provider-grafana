resource "grafana_synthetic_monitoring_probe" "main" {
  name      = "Mount Everest"
  latitude  = 27.98606
  longitude = 86.92262
  region    = "APAC"
  labels = {
    type = "mountain"
  }
}
