resource "grafana_assistant_watcher" "checkout" {
  name                     = "Checkout health"
  prompt                   = "Monitor checkout errors and availability. Treat routine deployments as expected changes."
  datasource_uids          = ["prometheus", "loki"]
  trigger_interval_seconds = 600
  sensitivity              = "balanced"
  labels = {
    service = "checkout"
  }
}

resource "grafana_assistant_watcher_calibration" "checkout" {
  watcher_id      = grafana_assistant_watcher.checkout.id
  started         = false
  timeout_seconds = 1200
}
