resource "grafana_slo" "search_expression" {
  name        = "Terraform Testing - Entity Search Expression"
  description = "Terraform Description - Entity Search Expression"
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

  search_expression = "Entity Search for RCA Workbench"
}
