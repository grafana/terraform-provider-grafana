terraform {
  required_providers {
    grafana = { 
      version = "0.2"
      source  = "registry.terraform.io/grafana/grafana"
    }
  }
}

provider "grafana" {
  url = "https://elainetest.grafana.net"
}

resource "grafana_slo_resource" "sample" {}

output "sample_slo" {
  value = grafana_slo_resource.sample
}
