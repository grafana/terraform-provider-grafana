resource "grafana_user" "test" {
  email    = "staff.name@example.com"
  name     = "Staff Name"
  login    = "staff"
  password = "my-password"
  is_admin = false
}

data "grafana_user" "from_id" {
  email = grafana_user.test.id
}

data "grafana_user" "from_email" {
  email = grafana_user.test.email
}

data "grafana_user" "from_login" {
  email = grafana_user.test.login
}
