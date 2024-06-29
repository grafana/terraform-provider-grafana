data "grafana_cloud_stack" "test" {
  slug = "gcloudstacktest"
}

data "grafana_cloud_provider_aws_account" "test" {
  stack_id    = data.grafana_cloud_stack.test.id
  resource_id = "1"
}