resource "grafana_organization" "test" {
  name         = "test-org"
  admin_user   = "admin"
  create_users = true
  viewers = [
    "viewer-01@example.com",
    "viewer-02@example.com",
  ]
}

data "grafana_organization" "from_name" {
  name = grafana_organization.test.name
}
