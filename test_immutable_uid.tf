terraform {
  required_providers {
    grafana = {
      source = "grafana/grafana"
    }
  }
}

provider "grafana" {
  url  = "http://localhost:3000"
  auth = "admin:admin"
}

# Step 1: Create a dashboard with initial UID
resource "grafana_dashboard" "test" {
  config_json = jsonencode({
    title = "Test Dashboard"
    uid   = "test-dashboard-1"
  })
}

output "dashboard_uid" {
  value = grafana_dashboard.test.uid
} 