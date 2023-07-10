resource "grafana_team" "test" {
  name  = "test-team"
  email = "test-team-email@test.com"

  preferences {
    theme    = "dark"
    timezone = "utc"
  }

  team_sync {
    groups = ["group1", "group2"]
  }
}

data "grafana_team" "from_name" {
  name           = grafana_team.test.name
  read_team_sync = true
}
