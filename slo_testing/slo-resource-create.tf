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

resource "grafana_slo" "test2" {
  name        = "Terraform - Name Test"
  description = "Terraform - Description Test"
  query {
    freeformquery = "sum(rate(apiserver_request_total{code!=\"500\"}[$__rate_interval])) / sum(rate(apiserver_request_total[$__rate_interval]))"
    type = "freeform"
  }
  objectives {
    objective_value  = 0.995
    objective_window = "30d"
  }
  labels {
    key   = "custom"
    value = "value"
  }
  alerting {
    fastburn {
      annotations {
        key   = "name"
        value = "Critical - SLO Burn Rate Alert"
      }
      annotations {
        key   = "description"
        value = "Error Budget Burning Very Quickly"
      }
      labels {
        key   = "type"
        value = "slo"
      }
    }

    slowburn {
      annotations {
        key   = "name"
        value = "Warning - SLO Burn Rate Alert"
      }
      annotations {
        key   = "description"
        value = "Error Budget Burning Quickly"
      }
      labels {
        key   = "type"
        value = "slo"
      }
    }
  }
}

output "test2" {
  value = grafana_slo.test2
}
