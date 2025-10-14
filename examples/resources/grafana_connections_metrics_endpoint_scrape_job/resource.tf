resource "grafana_connections_metrics_endpoint_scrape_job" "test" {
  stack_id                      = "1"
  name                          = "my-scrape-job"
  enabled                       = true
  authentication_method         = "basic"
  authentication_basic_username = "my-username"
  authentication_basic_password = "my-password"
  url                           = "https://grafana.com/metrics"
  scrape_interval_seconds       = 120
}
