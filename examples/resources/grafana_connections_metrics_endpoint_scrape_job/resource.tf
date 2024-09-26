resource "grafana_connections_metrics_endpoint_scrape_job" "test" {
  stack_id                      = "test-stack-id"
  name                          = "my-scrape-job"
  authentication_method         = "basic"
  authentication_basic_username = "my_username"
  authentication_basic_password = "my_password"
  url                           = "https://dev.my-metrics-endpoint-url.com:9000/metrics"
}
