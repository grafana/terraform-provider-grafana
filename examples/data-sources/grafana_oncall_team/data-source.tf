data "grafana_team" "my_team" {
  name = "my team"
}

data "grafana_oncall_team" "my_team" {
  name = data.grafana_team.my_team.name
}
