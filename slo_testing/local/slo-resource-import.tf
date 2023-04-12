# terraform {
#   required_providers {
#     grafana = { 
#       version = "0.2"
#       source  = "registry.terraform.io/grafana/grafana"
#     }
#   }
# }

# provider "grafana" {
#   url = "http://localhost:3000/api/plugins/grafana-slo-app/resources/v1/slo"
#   auth = "auth"
# }

# resource "grafana_slo_resource" "sample" {}

# output "sample_slo" {
#   value = grafana_slo_resource.sample
# }
