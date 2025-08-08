# Basic suppressed alert configuration for maintenance
resource "grafana_asserts_suppressed_assertions_config" "maintenance_window" {
  name = "MaintenanceWindow"

  match_labels = {
    service     = "api-service"
    maintenance = "true"
  }
}

# Suppress specific alertname during deployment
resource "grafana_asserts_suppressed_assertions_config" "deployment_suppression" {
  name = "DeploymentSuppression"

  match_labels = {
    alertname = "HighLatency"
    job       = "web-service"
    env       = "staging"
  }
}

# Suppress alerts for specific test environment
resource "grafana_asserts_suppressed_assertions_config" "test_environment_suppression" {
  name = "TestEnvironmentSuppression"

  match_labels = {
    alertgroup  = "test.alerts"
    environment = "test"
  }
}
