// Create the team with Grafana
resource "grafana_team" "example_team" {
  name  = "Example Team"
  email = "my-test-email@example.com"
}

// Get the OnCall-specific ID of the team with the datasource
data "grafana_oncall_team" "example_team" {
  name = grafana_team.example_team.name
}
