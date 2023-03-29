terraform {
  required_providers {
    grafana = {
      version = "0.2"
      source  = "registry.terraform.io/grafana/grafana"
    }
  }
}

provider "grafana" {
  auth = "auth"
}

data "grafana_slo" "test" {
}

output "test" {
  value = data.grafana_slo.test
}