resource "grafana_apps_dashboard_dashboard_v1beta1" "example" {
  metadata {
    uid = "example-dashboard"
  }

  spec {
    title = "Example Dashboard"
    json = jsonencode({
      title         = "Example Dashboard"
      uid           = "example-dashboard"
      panels        = []
      schemaVersion = 42
    })
  }
}
