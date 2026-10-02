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

  # A Knowledge Graph search expression scoping this SLO to a set of entities.
  # Setting it adds an "Open RCA workbench" link to the SLO list and performance
  # pages, and makes generated burn-rate alert rules carry a
  # `workbench_troubleshoot_url` annotation pre-filtered to those entities.
  search_expression = "shipping connected services"
}
