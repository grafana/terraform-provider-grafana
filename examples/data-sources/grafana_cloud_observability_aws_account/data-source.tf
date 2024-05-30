data "grafana_cloud_stack" "test" {
  slug = grafana_cloud_stack.test.slug
}

data "grafana_cloud_observability_aws_account" "test" {
  stack_id = data.grafana_cloud_stack.test.id
  name     = "my-aws-account"
}
