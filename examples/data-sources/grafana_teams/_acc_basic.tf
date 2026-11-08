resource "grafana_team" "dev_alpha" {
  name = "acc-test-dev-alpha"
}

resource "grafana_team" "dev_beta" {
  name = "acc-test-dev-beta"
}

resource "grafana_team" "ops_gamma" {
  name = "acc-test-ops-gamma"
}

data "grafana_teams" "by_query" {
  query = "acc-test-dev"

  depends_on = [
    grafana_team.dev_alpha,
    grafana_team.dev_beta,
    grafana_team.ops_gamma,
  ]
}
