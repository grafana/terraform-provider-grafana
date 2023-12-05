resource "grafana_dashboard" "metrics" {
  config_json = file("grafana-dashboard.json")
}

resource "grafana_team" "team" {
  name = "Team Name"
}

resource "grafana_team_preferences" "team_preferences" {
  team_id            = grafana_team.team.id
  theme              = "dark"
  timezone           = "browser"
  home_dashboard_uid = grafana_dashboard.metrics.uid
}
