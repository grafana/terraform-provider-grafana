resource "grafana_user" "viewer" {
  name     = "Viewer"
  email    = "viewer@example.com"
  login    = "viewer"
  password = "my-password"
}

resource "grafana_team" "test-team" {
  name  = "Test Team"
  email = "teamemail@example.com"
  members = [
    grafana_user.viewer.email,
  ]
}
