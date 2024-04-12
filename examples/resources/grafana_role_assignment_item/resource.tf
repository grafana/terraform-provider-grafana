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

resource "grafana_role_assignment_item" "user" {
  role_uid = grafana_role.test_role.uid
  user_id  = grafana_user.test_user.id
}

resource "grafana_role_assignment_item" "team" {
  role_uid = grafana_role.test_role.uid
  team_id  = grafana_team.test_team.id
}

resource "grafana_role_assignment_item" "service_account" {
  role_uid           = grafana_role.test_role.uid
  service_account_id = grafana_service_account.test_sa.id
}
