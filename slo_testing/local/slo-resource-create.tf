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
#     key   = "name"
#     value = "testslolabel"
#   }
#   alerting {
#     name = "hihialerting1"
#     labels {
#       key   = "name"
#       value = "testsloalertinglabel"
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

# resource "grafana_slo_resource" "test2" {
#   name        = "Hello2"
#   description = "Testing Hello 2 - I hope this works!"
#   service     = "service2"
#   query       = "sum(rate(apiserver_request_total{code!=\"500\"}[$__rate_interval])) / sum(rate(apiserver_request_total[$__rate_interval]))"
#   objectives {
#     objective_value  = 0.85
#     objective_window = "28d"
#   }
#   labels {
#     key   = "label2a"
#     value = "value2a"
#   }
#   labels {
#     key   = "label3a"
#     value = "value3a"
#   }
#   alerting {
#     name = "hihialerting2"
#     labels {
#       key   = "alertinglabel2"
#       value = "alertingvalue2"
#     }

#     annotations {
#       key   = "alertingannot2"
#       value = "alertingvalue2"
#     }

#     fastburn {
#       labels {
#         key   = "labelsfastburnkey2"
#         value = "labelsfastburnvalue2"
#       }
#       annotations {
#         key   = "annotsfastburnannot2"
#         value = "annotsfastburnvalue2"
#       }
#     }

#     slowburn {
#       labels {
#         key   = "labelsslowburnkey2"
#         value = "labelsslowburnvalue2"
#       }
#       annotations {
#         key   = "annotsslowburnannot2"
#         value = "annotsslowburnvalue2"
#       }
#     }
#   }
# }

# output "test2_order" {
#   value = grafana_slo_resource.test2
# }