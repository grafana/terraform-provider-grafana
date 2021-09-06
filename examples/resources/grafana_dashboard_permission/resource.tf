resource "grafana_team" "team" {
  name = "Team Name"
}

resource "grafana_user" "user" {
  email = "user.name@example.com"
}

resource "grafana_dashboard" "metrics" {
  config_json = file("grafana-dashboard.json")
}

resource "grafana_dashboard_permission" "collectionPermission" {
  dashboard_id = grafana_dashboard.metrics.dashboard_id
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
