terraform {
  required_version = ">= 1.5"

  required_providers {
    grafana = {
      source  = "grafana/grafana"
      version = "4.41.0"
    }
  }
}

provider "grafana" {}

# Fixed name: safe with equivalence tests (fresh Grafana each run via docker compose).
resource "grafana_team" "equivalence" {
  name  = "terraform-equivalence-grafana-team"
  email = "terraform-equivalence-grafana-team@example.com"
}
