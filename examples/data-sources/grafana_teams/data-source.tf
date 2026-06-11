data "grafana_teams" "all" {}

data "grafana_teams" "dev" {
  query = "dev"
}
