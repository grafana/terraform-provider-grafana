resource "grafana_data_source" "source_prometheus" {
  type = "prometheus"
  name = "SLO Source Prometheus"
  url  = "https://prometheus.example.com/"
}

resource "grafana_slo" "source_datasource" {
  name        = "Terraform Testing - Separate Source Datasource"
  description = "Terraform Description - Separate Source Datasource"
  query {
    freeform {
      query                 = "sum(rate(apiserver_request_total{code!=\"500\"}[$__rate_interval])) / sum(rate(apiserver_request_total[$__rate_interval]))"
      source_datasource_uid = grafana_data_source.source_prometheus.uid
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
}
