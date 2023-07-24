resource "grafana_slo" "ratio" {
  name        = "Terraform Testing - Ratio Query"
  description = "Terraform Description - Ratio Query"
  query {
    ratio {
      success_metric = "kubelet_http_requests_total{status!~\"5..\"}"
      total_metric   = "kubelet_http_requests_total"
      group_by_labels = ["job","instance"]
    }
    type          = "ratio"
  }
  objectives {
    value  = 0.995
    window = "30d"
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