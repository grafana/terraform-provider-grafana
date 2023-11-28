resource "grafana_slo" "test" {
  name        = "Terraform Testing"
  description = "Terraform Description"
  query {
    freeform {
      query = "sum(rate(apiserver_request_total{code!=\"500\"}[$__rate_interval])) / sum(rate(apiserver_request_total[$__rate_interval]))"
    }
    type = "freeform"
  }
  objectives {
    value  = 0.995
    window = "30d"
  }
  destination_datasource {
    uid = "grafanacloud-prom"
  }
  label {
    key   = "custom"
    value = "value"
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

data "grafana_slos" "slos" {}