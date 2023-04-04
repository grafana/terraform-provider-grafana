# Testing the UPDATE Method 
# 1. When testing the UPDATE Method, start up the Local Dev Environment. Comment out all the other `.tf` files. 
# 2. Within the root directory of the terraform-provider-grafana, run `make install`. This creates a Grafana Terraform Provider.
# 3. Switch to the slo_testing directory `cd slo_testing`
# 4. Run the command `terraform init`
# 5. Run the command `terraform apply`. This creates the resource specified below. 
# 6. To ensure that the PUT endpoint works, modify any of the values within the resource below, and re-run `terraform apply`. 
# 7. Using Postman, send a GET Request to the endpoint to ensure that the resource was appropriately modified within the API. 

# terraform {
#   required_providers {
#     grafana = { 
#       version = "0.2"
#       source  = "registry.terraform.io/grafana/grafana"
#     }
#   }
# }

# provider "grafana" {
#   auth = "auth"
# }

# resource "grafana_slo_resource" "test1" {
#   name        = "Hello1"
#   description = "Testing Hello 1 - I hope this works!"
#   service     = "service1"
#   query       = "sum(rate(apiserver_request_total{code!=\"500\"}[$__rate_interval])) / sum(rate(apiserver_request_total[$__rate_interval]))"
#   objectives {
#     objective_value  = 0.85
#     objective_window = "28d"
#   }
#   labels {
#     key   = "label1a"
#     value = "value1a"
#   }
#   labels {
#     key   = "label2a"
#     value = "value2a"
#   }
#   alerting {
#     name = "hihialerting1"
#     labels {
#       key   = "alertinglabel1"
#       value = "alertingvalue1"
#     }

#     annotations {
#       key   = "alertingannot1"
#       value = "alertingvalue1"
#     }

#     fastburn {
#       labels {
#         key   = "labelsfastburnkey1"
#         value = "labelsfastburnvalue1"
#       }
#       annotations {
#         key   = "annotsfastburnannot1"
#         value = "annotsfastburnvalue1"
#       }
#     }

#     slowburn {
#       labels {
#         key   = "labelsslowburnkey1"
#         value = "labelsslowburnvalue1"
#       }
#       annotations {
#         key   = "annotsslowburnannot1"
#         value = "annotsslowburnvalue1"
#       }
#     }
#   }
# }

# output "test1_order" {
#   value = grafana_slo_resource.test1
# }