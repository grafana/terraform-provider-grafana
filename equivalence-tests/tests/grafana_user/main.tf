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

# Instance-scoped admin APIs: requires basic auth (not API tokens). Default
# equivalence targets set GRAFANA_AUTH to admin:admin.
#
# Fixed login/email: each run uses a fresh temp dir, so Terraform always tries
# to create this user again. Delete the user first if you see HTTP 409, or run
# `make equivalence-test-delete-user`.
resource "grafana_user" "equivalence" {
  email    = "terraform-equivalence-grafana-user@example.com"
  name     = "Terraform Equivalence User"
  login    = "terraform-equiv-grafana-user"
  password = "equivalence-test-password-do-not-reuse"
  is_admin = false
}
