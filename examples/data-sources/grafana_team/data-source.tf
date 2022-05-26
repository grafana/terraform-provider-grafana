resource "grafana_team" "test" {
  name  = "test-team"
  email = "test-team-email@test.com"
}

data "grafana_team" "from_name" {
  name = grafana_team.test.name
}
