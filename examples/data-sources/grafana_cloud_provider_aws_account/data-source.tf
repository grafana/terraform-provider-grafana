data "grafana_cloud_stack" "test" {
  slug = "gcloudstacktest"
}

resource "grafana_cloud_provider_aws_account" "test" {
  stack_id = data.grafana_cloud_stack.test.id
  role_arn = data.aws_iam_role.test.arn
  regions = [
    "us-east-2",
    "eu-west-3"
  ]
}

data "grafana_cloud_provider_aws_account" "test" {
  stack_id    = data.grafana_cloud_stack.test.id
  resource_id = grafana_cloud_provider_aws_account.test.resource_id
}
