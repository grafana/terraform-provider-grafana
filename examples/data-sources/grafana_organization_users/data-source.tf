resource "grafana_organization" "test" {
  name         = "test-org"
  create_users = true
  admin_user   = "admin"
  admins = [
    "admin@example.com"
  ]
}

data "grafana_organization_users" "from_name" {
  organization_name = grafana_organization.test_org.name
}
