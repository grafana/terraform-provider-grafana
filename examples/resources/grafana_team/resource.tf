resource "grafana_user" "viewer" {
  name     = "Viewer"
  email    = "viewer@example.com"
  login    = "viewer"
  password = "my-password"
}

resource "grafana_user" "editor" {
  name     = "Editor"
  email    = "editor@example.com"
  login    = "editor"
  password = "my-password-2"
}

resource "grafana_team" "test-team" {
  name  = "Test Team"
  email = "teamemail@example.com"
  members = [
    grafana_user.viewer.email,
  ]
  admins = [
    grafana_user.editor.email,
  ]
}
