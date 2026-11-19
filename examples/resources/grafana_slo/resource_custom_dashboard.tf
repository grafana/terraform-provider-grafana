resource "grafana_slo" "custom_dashboard" {
  name        = "Terraform Testing - Custom Dashboard"
  description = "Terraform Description - Custom Dashboard"
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
    key   = "slo"
    value = "terraform"
  }

  # The UID of an existing Grafana dashboard to use as this SLO's drill-down
  # dashboard instead of the one generated for the SLO. Dashboard links and
  # generated alert rule annotation URLs point at it. The dashboard is not
  # created or managed by the SLO, and its existence is not checked.
  custom_dashboard_uid = "my-service-overview"
}
