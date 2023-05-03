terraform {
  required_providers {
    grafana = {
      source  = "registry.terraform.io/grafana/grafana"
    }
  }
}

provider "grafana" {
  url = "https://elainetest.grafana.net/"
}

data "grafana_slo" "test1" {
}

output "test1" {
  value = data.grafana_slo.test1
}