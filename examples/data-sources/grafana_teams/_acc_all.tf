resource "grafana_team" "test_one" {
  name = "acc-test-teams-one"
}

resource "grafana_team" "test_two" {
  name = "acc-test-teams-two"
}

resource "grafana_team" "test_three" {
  name = "acc-test-teams-three"
}

data "grafana_teams" "all" {
  depends_on = [
    grafana_team.test_one,
    grafana_team.test_two,
    grafana_team.test_three,
  ]
}
