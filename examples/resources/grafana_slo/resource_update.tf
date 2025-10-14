resource "grafana_folder" "folder" {
  title = "Terraform Testing"
}

resource "grafana_slo" "test" {
  name        = "Updated - Terraform Testing"
  description = "Updated - Terraform Description"
  folder_uid  = grafana_folder.folder.uid
  query {
    freeform {
      query = "sum(rate(apiserver_request_total{code!=\"500\"}[$__rate_interval])) / sum(rate(apiserver_request_total[$__rate_interval]))"
    }
    type = "freeform"
  }
  objectives {
    value  = 0.9995
    window = "7d"
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
  }
}
