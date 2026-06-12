terraform {
  required_version = ">= 1.5"

  required_providers {
    grafana = {
      source  = "grafana/grafana"
      version = "4.38.0"
    }
  }
}

provider "grafana" {}

resource "grafana_user" "equivalence" {
  email    = "terraform-equivalence-grafana-user@example.com"
  name     = "Terraform Equivalence User"
  login    = "terraform-equiv-grafana-user"
  password = "equivalence-test-password-do-not-reuse"
  is_admin = false
}
