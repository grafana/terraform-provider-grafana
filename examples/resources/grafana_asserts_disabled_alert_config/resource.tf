resource "grafana_asserts_disabled_alert_config" "maintenance_window" {
  name = "MaintenanceWindow"

  match_labels = {
    service     = "api-service"
    maintenance = "true"
  }
} 