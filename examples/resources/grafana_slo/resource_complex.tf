resource "grafana_slo" "test" {
  name        = "Complex Resource - Terraform Testing"
  description = "Complex Resource - Terraform Description"
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
  label {
    key   = "slokey"
    value = "slokey"
  }
  alerting {
    label {
      key   = "alertingkey"
      value = "alertingvalue"
    }

    fastburn {
      annotation {
        key   = "name"
        value = "Critical - SLO Burn Rate Alert - {{$labels.instance}}"
      }
      annotation {
        key   = "description"
        value = "Error Budget is burning at a rate greater than 14.4x."
      }
      label {
        key   = "type"
        value = "slo"
      }
    }

    slowburn {
      annotation {
        key   = "name"
        value = "Warning - SLO Burn Rate Alert - {{$labels.instance}}"
      }
      annotation {
        key   = "description"
        value = "Error Budget is burning at a rate greater than 1x."
      }
      label {
        key   = "type"
        value = "slo"
      }
    }
  }
}