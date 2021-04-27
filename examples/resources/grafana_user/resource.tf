resource "grafana_user" "staff" {
  email    = "staff.name@example.com"
  name     = "Staff Name"
  login    = "staff"
  password = "my-password"
  is_admin = false
}
