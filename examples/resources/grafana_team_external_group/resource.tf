resource "grafana_team_external_group" "test-team-group" {
  team_id = 1
  groups = [
    "test-group-1",
    "test-group-2"
  ]
}