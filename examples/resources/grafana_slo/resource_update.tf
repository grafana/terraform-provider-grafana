resource "grafana_slo_resource" "update" {
  name        = "Modified - Terraform Testing"
  description = "Modified - Terraform Description"
  query       = "sum(rate(apiserver_request_total{code!=\"500\"}[$__rate_interval])) / sum(rate(apiserver_request_total[$__rate_interval]))"
  objectives {
    objective_value  = 0.9995
    objective_window = "7d"
  }
  labels {
    key   = "customkey"
    value = "customvalue"
  }
  alerting {
    fastburn {
      annotations {
        key   = "name"
        value = "Critical - SLO Burn Rate Alert"
      }
      annotations {
        key   = "description"
        value = "Error Budget is burning at a rate greater than 14.4x."
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
        value = "Error Budget is burning at a rate greater than 1x."
      }
      labels {
        key   = "type"
        value = "slo"
      }
    }
  }
}