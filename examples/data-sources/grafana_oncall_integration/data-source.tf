data "grafana_oncall_integration" "example_integration" {
  id = "CEXAMPLEID123"
}

data "grafana_oncall_team" "my_team" {
  name = "my_team"
}

# The direct paging integration of a team, without its ID.
data "grafana_oncall_integration" "my_team_direct_paging" {
  team_id = data.grafana_oncall_team.my_team.id
  type    = "direct_paging"
}

data "grafana_oncall_integration" "by_name" {
  name = "My integration"
}
