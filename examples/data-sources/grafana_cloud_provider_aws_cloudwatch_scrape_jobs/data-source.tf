data "grafana_cloud_stack" "test" {
  slug = "gcloudstacktest"
}

data "grafana_cloud_provider_aws_cloudwatch_scrape_jobs" "test" {
  stack_id = data.grafana_cloud_stack.test.id
}
