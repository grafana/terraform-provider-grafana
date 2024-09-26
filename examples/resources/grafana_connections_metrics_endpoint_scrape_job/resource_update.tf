resource "grafana_connections_metrics_endpoint_scrape_job" "test" {
  stack_id                    = "test-stack-id"
  name                        = "modified-scrape-job"
  enabled                     = "false"
  authentication_method       = "bearer"
  authentication_bearer_token = "test-token"
  url                         = "https://www.modified-url.com:9000/metrics"
  scrape_interval_seconds     = "120"
}
