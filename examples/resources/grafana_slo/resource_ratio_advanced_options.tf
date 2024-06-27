resource "grafana_slo" "ratio_options" {
  name        = "Terraform Testing - Ratio Query"
  description = "Terraform Description - Ratio Query"
  query {
    ratio {
      success_metric  = "kubelet_http_requests_total{status!~\"5..\"}"
      total_metric    = "kubelet_http_requests_total"
      group_by_labels = ["job", "instance"]
    }
    type = "ratio"
  }
  objectives {
    value  = 0.995
    window = "30d"
  }
  destination_datasource {
    uid = "grafanacloud-prom"
  }
  label {
    key   = "slo"
    value = "terraform"
  }
  alerting {
    fastburn {
      annotation {
        key   = "name"
        value = "SLO Burn Rate Very High"
      }
      annotation {
        key   = "description"
        value = "Error budget is burning too fast"
      }
    }

    slowburn {
      annotation {
        key   = "name"
        value = "SLO Burn Rate High"
      }
      annotation {
        key   = "description"
        value = "Error budget is burning too fast"
      }
    }
    advanced_options {
      min_failures = 10
    }
  }
}