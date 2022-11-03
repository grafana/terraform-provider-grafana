resource "grafana_role" "test_role" {
  name    = "Test Role"
  uid     = "testrole"
  version = 1
  global  = true

  permissions {
    action = "org.users:add"
    scope  = "users:*"
  }
}

resource "grafana_team" "test_team" {
  name = "terraform_test_team"
}

resource "grafana_user" "test_user" {
  email    = "terraform_user@test.com"
  login    = "terraform_user@test.com"
  password = "password"
}

resource "grafana_service_account" "test_sa" {
  name = "terraform_test_sa"
  role = "Viewer"
}

resource "grafana_role_assignment" "test" {
  role_uid         = grafana_role.test_role.uid
  users            = [grafana_user.test_user.id]
  teams            = [grafana_team.test_team.id]
  service_accounts = [grafana_service_account.test_sa.id]
}
