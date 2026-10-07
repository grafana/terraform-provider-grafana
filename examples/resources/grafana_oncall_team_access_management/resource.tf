resource "grafana_team" "example" {
  name = "Example team"
}

data "grafana_oncall_team" "example" {
  name       = grafana_team.example.name
  depends_on = [grafana_team.example]
}

// Restrict the team's OnCall resources to team members and admins
resource "grafana_oncall_team_access_management" "example" {
  team_id                     = data.grafana_oncall_team.example.id
  is_sharing_resources_to_all = false
}
