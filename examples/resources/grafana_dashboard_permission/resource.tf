resource "grafana_team" "team" {
  name = "Team Name"
}

resource "grafana_user" "user" {
  email    = "user.name@example.com"
  password = "my-password"
  login    = "user.name"
}

resource "grafana_dashboard" "metrics" {
  config_json = jsonencode({
    "title" : "My Dashboard",
    "uid" : "my-dashboard-uid"
  })
}

resource "grafana_dashboard_permission" "collectionPermission" {
  dashboard_uid = grafana_dashboard.metrics.uid
  permissions {
    role       = "Editor"
    permission = "Edit"
  }
  permissions {
    team_id    = grafana_team.team.id
    permission = "View"
  }
  permissions {
    user_id    = grafana_user.user.id
    permission = "Admin"
  }
}
