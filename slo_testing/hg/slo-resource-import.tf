terraform {
  required_providers {
    grafana = {
      source = "registry.terraform.io/grafana/grafana"
    }
  }
}

provider "grafana" {
  url = "https://elainetest.grafana.net"
}

resource "grafana_slo_resource" "sample" {
  name        = "Terraform - Import Test123"
  description = "Terraform - Import Test"
  query       = "sum(rate(apiserver_request_total{code!=\"500\"}[$__rate_interval])) / sum(rate(apiserver_request_total[$__rate_interval]))"
  objectives {
    objective_value  = 0.995
    objective_window = "30d"
  }
}

output "sample_slo" {
  value = grafana_slo_resource.sample
}
