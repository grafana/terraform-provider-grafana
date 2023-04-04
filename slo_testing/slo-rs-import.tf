# terraform import grafana_slo_resource.sample 7jglyd0d965pbmbikv62l
# terraform state show grafana_slo_resource.sample

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

resource "grafana_slo_resource" "sample" {}

output "sample_slo" {
  value = grafana_slo_resource.sample
}

