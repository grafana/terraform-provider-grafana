resource "grafana_user" "viewer" {
  name     = "Viewer"
  email    = "viewer@example.com"
  login    = "viewer"
  password = "my-password"
}

resource "grafana_user" "team_admin" {
  name     = "Team Admin"
  email    = "team-admin@example.com"
  login    = "team-admin"
  password = "my-password-2"
}

resource "grafana_team" "test-team" {
  name  = "Test Team"
  email = "teamemail@example.com"
  members = [
    grafana_user.viewer.email,
  ]
  admins = [
    grafana_user.team_admin.email,
  ]
}
