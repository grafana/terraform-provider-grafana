terraform {
  required_version = ">= 1.5"

  required_providers {
    grafana = {
      source  = "grafana/grafana"
      version = "4.35.0"
    }
  }
}

provider "grafana" {}

# Fixed name: safe with `make equivalence-test-*-docker` (fresh Grafana each run).
resource "grafana_team" "equivalence" {
  name  = "terraform-equivalence-grafana-team"
  email = "terraform-equivalence-grafana-team@example.com"
}
