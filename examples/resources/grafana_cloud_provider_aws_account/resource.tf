data "grafana_cloud_stack" "test" {
  slug = "gcloudstacktest"
}

data "aws_iam_role" "test" {
  name = "my-role"
}

resource "grafana_cloud_provider_aws_account" "test" {
  stack_id = data.grafana_cloud_stack.test.id
  role_arn = data.aws_iam_role.test.arn
  regions = [
    "us-east-1",
    "us-east-2",
    "us-west-1"
  ]
}
