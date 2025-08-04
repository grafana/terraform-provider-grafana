resource "grafana_asserts_disabled_alert_config" "maintenance_window" {
  stack_id = data.grafana_cloud_stack.test.id
  name     = "MaintenanceWindow"

  match_labels = {
    service     = "api-service"
    maintenance = "true"
  }
} 