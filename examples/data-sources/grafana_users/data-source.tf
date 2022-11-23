resource "grafana_user" "test_all_users" {
  email    = "all_users@example.com"
  name     = "Testing grafana_users"
  login    = "test-grafana-users"
  password = "my-password"
}

data "grafana_users" "all_users" {
  depends_on = [
    grafana_user.test_all_users,
  ]
}
