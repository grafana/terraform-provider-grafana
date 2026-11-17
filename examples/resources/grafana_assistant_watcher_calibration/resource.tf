resource "grafana_assistant_watcher" "alerts" {
  name   = "Managed alerts"
  prompt = "Monitor Grafana-managed firing alerts. An empty result is an acceptable baseline."
}

# Setup is separate so a timeout cannot taint or replace the Watcher itself.
resource "grafana_assistant_watcher_calibration" "alerts" {
  watcher_id      = grafana_assistant_watcher.alerts.id
  started         = false
  timeout_seconds = 1200
}
