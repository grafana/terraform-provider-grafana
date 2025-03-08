resource "grafana_user" "test" {
  email    = "test.datasource@example.com"
  name     = "Testing Datasource"
  login    = "test-datasource"
  password = "my-password"
}

data "grafana_organization_user" "from_email" {
  email = grafana_user.test.email
}

data "grafana_organization_user" "from_login" {
  login = grafana_user.test.login
}
