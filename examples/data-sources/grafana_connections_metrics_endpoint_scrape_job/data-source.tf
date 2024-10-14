resource "grafana_connections_metrics_endpoint_scrape_job" "test" {
  stack_id              = "1"
  name                  = "scrape-job-name"
  authentication_method = "basic"
  authentication_basic_username = "my-username"
  authentication_basic_password = "my-password"
  url                   = "https://dev.my-metrics-endpoint-url.com:9000/metrics"
}

data "grafana_connections_metrics_endpoint_scrape_job" "test" {
  stack_id              = grafana_connections_metrics_endpoint_scrape_job.test.stack_id
  name                  = grafana_connections_metrics_endpoint_scrape_job.test.name
  authentication_method = grafana_connections_metrics_endpoint_scrape_job.test.authentication_method
  url                   = grafana_connections_metrics_endpoint_scrape_job.test.url
}
