data "grafana_connections_metrics_endpoint_scrape_job" "ds_test" {
  stack_id = "1"
  name     = "my-scrape-job"
}
