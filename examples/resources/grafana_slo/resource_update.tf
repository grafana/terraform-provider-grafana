resource "grafana_slo" "update" {
  name        = "Updated - Terraform Testing"
  description = "Updated - Terraform Description"
  query {
    freeform {
      query = "sum(rate(apiserver_request_total{code!=\"500\"}[$__rate_interval])) / sum(rate(apiserver_request_total[$__rate_interval]))"
    }
    type = "freeform"
  }
  objectives {
    value  = 0.9995
    window = "7d"
  }
  label {
    key   = "slokey"
    value = "slovalue"
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