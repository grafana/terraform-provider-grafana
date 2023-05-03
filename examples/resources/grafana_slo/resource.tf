resource "grafana_slo" "test" {
  name        = "Terraform Testing"
  description = "Terraform Description"
  query {
    freeformquery = "sum(rate(apiserver_request_total{code!=\"500\"}[$__rate_interval])) / sum(rate(apiserver_request_total[$__rate_interval]))"
    type          = "freeform"
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
      labels {
        key   = "type"
        value = "slo"
      }
    }
  }
}