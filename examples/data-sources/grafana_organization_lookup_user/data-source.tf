resource "grafana_user" "test" {
  email    = "test.datasource@example.com"
  name     = "Testing Datasource"
  login    = "test-datasource"
  password = "my-password"
}

data "grafana_organization_lookup_user" "test" {
  login = grafana_user.test.login
}
