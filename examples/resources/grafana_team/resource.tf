resource "grafana_team" "test-team" {
  name  = "Test Team"
  email = "teamemail@example.com"
  members = [
    "viewer-01@example.com"
  ]
}
