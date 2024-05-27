resource "grafana_data_source" "prometheus" {
  name = "Terraform Testing"
  type = "prometheus"
  url  = "http://localhost:9090"
}

resource "grafana_slo" "test" {
  name        = "Complex Resource - Terraform Ratio Query Example"
  description = "Complex Resource - Terraform Ratio Query Description"
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
    uid = grafana_data_source.prometheus.uid
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
      label {
        key   = "type"
        value = "slo"
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
      label {
        key   = "type"
        value = "slo"
      }
    }
  }
}
