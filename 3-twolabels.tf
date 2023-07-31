terraform {
  required_providers {
    grafana = { 
      version = "0.2"
      source  = "registry.terraform.io/grafana/grafana"
    }
  }
}

provider "grafana" {
  url = "http://localhost:3000"
}

resource "grafana_slo" "twolabel" {
  name        = "Terraform Testing - with one labels"
  description = "Terraform Description"
  query {
    type          = "freeform"
    freeform {
        query = "sum(rate(apiserver_request_total{code!=\"500\"}[$__rate_interval])) / sum(rate(apiserver_request_total[$__rate_interval]))"
    }
  }
  objectives {
    value  = 0.995
    window = "30d"
  }
  alerting {
    fastburn {
      annotation {
        key   = "name"
        value = "SLO Burn Rate Very High"
      }
      annotation {
        key   = "name"
        value = "Error budget is burning too fast"
      }
    }

    slowburn {
      annotation {
        key   = "name"
        value = "SLO Burn Rate High"
      }
      annotation {
        key   = "name"
        value = "Error budget is burning too fast"
      }
    }
  }
}

output "onelabel_order" {
  value = grafana_slo.twolabel
}