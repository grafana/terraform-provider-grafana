resource "grafana_connections_metrics_endpoint_scrape_job" "test" {
  stack_id                      = "1"
  name                          = "scrape-job-name"
  authentication_method         = "basic"
  authentication_basic_username = "my-username"
  authentication_basic_password = "my-password"
  url                           = "https://dev.my-metrics-endpoint-url.com:9000/metrics"
}
