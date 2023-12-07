resource "grafana_team" "my_team" {
  name = "My Team"
}

resource "grafana_team_external_group" "test-team-group" {
  team_id = grafana_team.my_team.id
  groups = [
    "test-group-1",
    "test-group-2"
  ]
}
