# 1. When testing the Create Method, start up the Local Dev Environment. Comment out the `slo-ds-read.tf` file. 
# 2. Within the root directory of the terraform-provider-grafana, run `make install`. This creates a Grafana Terraform Provider.
# 3. Switch to the slo_testing directory `cd slo_testing`
# 4. Run the command `terraform init`
# 5. Run the command `terraform apply slo-create.tf`. This sends the information below as a POST request to the API at http://grafana.k3d.localhost:3000/api/plugins/grafana-slo-app/resources/v1/slo
# 6. Ensure to delete the `.terraform.lock.hcl` and any hidden terraform (terraform.tfstate) state files that exists before rebuilding the terraform provider. 
# 7. To determine that the new creation was successful, uncomment out lines 76-81, and run `terraform apply` again. 
# 8. Within your terminal, you should see the output of the newly created SLO from within Terraform (should match the output from executing a GET to http://grafana.k3d.localhost:3000/api/plugins/grafana-slo-app/resources/v1/slo)

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

resource "grafana_slo_resource" "test1" {
  name        = "Hello1"
  description = "Testing Hello 1 - I hope this works!"
  service     = "service1"
  query       = "sum(rate(apiserver_request_total{code!=\"500\"}[$__rate_interval])) / sum(rate(apiserver_request_total[$__rate_interval]))"
  objectives {
    objective_value  = 0.85
    objective_window = "28d"
  }
  labels {
    key   = "label1a"
    value = "value1a"
  }
  labels {
    key   = "label2a"
    value = "value2a"
  }
  alerting {
    name = "hihialerting1"
    labels {
      key   = "alertinglabel1"
      value = "alertingvalue1"
    }

    annotations {
      key   = "alertingannot1"
      value = "alertingvalue1"
    }

    fastburn {
      labels {
        key   = "labelsfastburnkey1"
        value = "labelsfastburnvalue1"
      }
      annotations {
        key   = "annotsfastburnannot1"
        value = "annotsfastburnvalue1"
      }
    }

    slowburn {
      labels {
        key   = "labelsslowburnkey1"
        value = "labelsslowburnvalue1"
      }
      annotations {
        key   = "annotsslowburnannot1"
        value = "annotsslowburnvalue1"
      }
    }
  }
}

resource "grafana_slo_resource" "test2" {
  name        = "Hello2"
  description = "Testing Hello 2 - I hope this works!"
  service     = "service2"
  query       = "sum(rate(apiserver_request_total{code!=\"500\"}[$__rate_interval])) / sum(rate(apiserver_request_total[$__rate_interval]))"
  objectives {
    objective_value  = 0.85
    objective_window = "28d"
  }
  labels {
    key   = "label2a"
    value = "value2a"
  }
  labels {
    key   = "label3a"
    value = "value3a"
  }
  alerting {
    name = "hihialerting2"
    labels {
      key   = "alertinglabel2"
      value = "alertingvalue2"
    }

    annotations {
      key   = "alertingannot2"
      value = "alertingvalue2"
    }

    fastburn {
      labels {
        key   = "labelsfastburnkey2"
        value = "labelsfastburnvalue2"
      }
      annotations {
        key   = "annotsfastburnannot2"
        value = "annotsfastburnvalue2"
      }
    }

    slowburn {
      labels {
        key   = "labelsslowburnkey2"
        value = "labelsslowburnvalue2"
      }
      annotations {
        key   = "annotsslowburnannot2"
        value = "annotsslowburnvalue2"
      }
    }
  }
}

data "grafana_slo_datasource" "test" {
}

output "test1" {
  value = data.grafana_slo_datasource.test
}