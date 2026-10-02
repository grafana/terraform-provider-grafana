resource "grafana_apps_queries_query_v1" "example" {
  metadata {
    uid = "example-saved-query"
  }

  spec {
    title       = "Requests per second"
    description = "Prometheus rate of HTTP requests"
    is_visible  = true
    tags        = ["http", "prometheus"]

    targets {
      properties_json = jsonencode({
        refId = "A"
        expr  = "rate(http_requests_total[$__rate_interval])"
        datasource = {
          type = "prometheus"
          uid  = "my-prometheus-uid"
        }
      })
    }
  }
}
