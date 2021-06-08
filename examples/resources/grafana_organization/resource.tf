resource "grafana_organization" "test" {
  name         = "Test Organization"
  admin_user   = "admin"
  create_users = true
  admins = [
    "admin@example.com"
  ]
  editors = [
    "editor-01@example.com",
    "editor-02@example.com"
  ]
  viewers = [
    "viewer-01@example.com",
    "viewer-02@example.com"
  ]
}
