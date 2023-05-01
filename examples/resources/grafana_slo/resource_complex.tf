resource "grafana_slo" "test" {
  name        = "Complex Resource - Terraform Testing"
  description = "Complex Resource - Terraform Description"
  query {
    query_type     = "freeform"
    freeform_query = "sum(rate(apiserver_request_total{code!=\"500\"}[$__rate_interval])) / sum(rate(apiserver_request_total[$__rate_interval]))"
  }
  objectives {
    value  = 0.995
    window = "30d"
  }
  labels {
    key   = "slokey"
    value = "slokey"
  }
  alerting {
    name = "alertingname"

    labels {
      key   = "alertingkey"
      value = "alertingvalue"
    }

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