resource "grafana_asserts_suppressed_assertions_config" "maintenance_window" {
  name = "MaintenanceWindow"

  match_labels = {
    service     = "api-service"
    maintenance = "true"
  }
} 