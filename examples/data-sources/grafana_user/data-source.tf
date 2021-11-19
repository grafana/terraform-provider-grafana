resource "grafana_user" "test" {
  email    = "test.datasource@example.com"
  name     = "Testing Datasource"
  login    = "test-datasource"
  password = "my-password"
  is_admin = true
}

data "grafana_user" "from_id" {
  user_id = grafana_user.test.user_id
}

data "grafana_user" "from_email" {
  email = grafana_user.test.email
}

data "grafana_user" "from_login" {
  login = grafana_user.test.login
}
