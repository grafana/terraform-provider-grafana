resource "grafana_service_account" "test" {
  name        = "terraform-sa"
  role        = "Editor"
  is_disabled = false
}

resource "grafana_team" "team" {
  name = "Team Name"
}

resource "grafana_user" "user" {
  email    = "user.name@example.com"
  login    = "user.name"
  password = "my-password"
}

resource "grafana_service_account_permission_item" "on_team" {
  service_account_id = grafana_service_account.test.id
  team               = grafana_team.team.id
  permission         = "Admin"
}

resource "grafana_service_account_permission_item" "on_user" {
  service_account_id = grafana_service_account.test.id
  user               = grafana_user.user.id
  permission         = "Admin"
}

