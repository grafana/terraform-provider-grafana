terraform {
  required_providers {
    grafana = {
      source = "registry.terraform.io/grafana/grafana"
    }
  }
}

provider "grafana" {
  url = "https://elainetest.grafana.net/"
}

resource "grafana_slo" "test1" {
  name        = "Terraform - Name Test"
  description = "Terraform - Description Test"
  query {
    query_type     = "freeform"
    freeform_query = "sum(rate(apiserver_request_total{code!=\"500\"}[$__rate_interval])) / sum(rate(apiserver_request_total[$__rate_interval]))"
  }
  objectives {
    value  = 0.995
    window = "30d"
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

output "test1" {
  value = grafana_slo.test1
}
