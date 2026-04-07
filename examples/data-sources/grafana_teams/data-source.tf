data "grafana_teams" "all" {}

output "team_names" {
  value = [for team in data.grafana_teams.all.teams : team.name]
}
