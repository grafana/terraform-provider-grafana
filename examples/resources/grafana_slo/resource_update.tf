resource "grafana_slo" "update" {
  name        = "Updated - Terraform Testing"
  description = "Updated - Terraform Description"
  query {
    freeformquery {
      query = "sum(rate(apiserver_request_total{code!=\"500\"}[$__rate_interval])) / sum(rate(apiserver_request_total[$__rate_interval]))"
    }
    type          = "freeform"
  }
  objectives {
    value  = 0.9995
    window = "7d"
  }
  label {
    key   = "customkey"
    value = "customvalue"
  }
  alerting {
    fastburn {
      annotation {
        key   = "name"
        value = "Critical - SLO Burn Rate Alert"
      }
      label {
        key   = "type"
        value = "slo"
      }
    }

    slowburn {
      annotation {
        key   = "name"
        value = "Warning - SLO Burn Rate Alert"
      }
      label {
        key   = "type"
        value = "slo"
      }
    }
  }
}