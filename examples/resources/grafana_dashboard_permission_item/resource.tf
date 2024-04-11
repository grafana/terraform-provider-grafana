resource "grafana_team" "team" {
  name = "Team Name"
}

resource "grafana_user" "user" {
  email    = "user.name@example.com"
  password = "my-password"
  login    = "user.name"
}

resource "grafana_dashboard" "dashboard" {
  config_json = jsonencode({
    "title" : "My Dashboard",
    "uid" : "my-dashboard-uid"
  })
}

resource "grafana_dashboard_permission_item" "role" {
  dashboard_uid = grafana_dashboard.dashboard.uid
  role          = "Viewer"
  permission    = "View"
}

resource "grafana_dashboard_permission_item" "user" {
  dashboard_uid = grafana_dashboard.dashboard.uid
  user          = grafana_user.user.id
  permission    = "Admin"
}

resource "grafana_dashboard_permission_item" "team" {
  dashboard_uid = grafana_dashboard.dashboard.uid
  team          = grafana_team.team.id
  permission    = "Edit"
}
