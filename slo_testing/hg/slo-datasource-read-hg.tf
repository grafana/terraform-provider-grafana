# terraform {
#   required_providers {
#     grafana = {
#       version = "0.2"
#       source  = "registry.terraform.io/grafana/grafana"
#     }
#   }
# }

# provider "grafana" {
#   url = "https://elainetest.grafana.net"
# }

# data "grafana_slo_datasource" "test1" {
# }

# output "test1" {
#   value = data.grafana_slo_datasource.test1
# }