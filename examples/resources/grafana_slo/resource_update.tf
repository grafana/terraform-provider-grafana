resource "grafana_slo" "update" {
  name        = "Updated - Terraform Testing"
  description = "Updated - Terraform Description"
  query {
    query_type = "freeform"
    freeform_query = "sum(rate(apiserver_request_total{code!=\"500\"}[$__rate_interval])) / sum(rate(apiserver_request_total[$__rate_interval]))"
  }
  objectives {
    value  = 0.9995
    window = "7d"
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