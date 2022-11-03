resource "grafana_service_account" "test" {
  name        = "sa-terraform-test"
  role        = "Editor"
  is_disabled = false
}

resource "grafana_team" "test_team" {
  name = "tf_test_team"
}

resource "grafana_user" "test_user" {
  email    = "tf_user@test.com"
  login    = "tf_user@test.com"
  password = "password"
}

resource "grafana_service_account_permission" "test_permissions" {
  service_account_id = grafana_service_account.test.id

  permissions {
    user_id    = grafana_user.test_user.id
    permission = "Edit"
  }
  permissions {
    team_id    = grafana_team.test_team.id
    permission = "Admin"
  }
}
