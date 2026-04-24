terraform {
  required_version = ">= 1.5"

  required_providers {
    grafana = {
      source  = "grafana/grafana"
      version = "999.999.999"
    }
  }
}

provider "grafana" {}

# Fixed name: each equivalence run uses a fresh temp dir, so Terraform always
# tries to create this team again. Use a clean Grafana org or delete the team
# before re-running update/diff.
resource "grafana_team" "equivalence" {
  name  = "terraform-equivalence-grafana-team"
  email = "terraform-equivalence-grafana-team@example.com"
}
