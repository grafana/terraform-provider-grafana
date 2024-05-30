data "grafana_cloud_stack" "test" {
  slug = grafana_cloud_stack.test.slug
}

resource "grafana_cloud_observability_aws_account" "test" {
  stack_id = data.grafana_cloud_stack.test.id
  name     = "my-aws-account"
  role_arns = {
    "my role 1a" = "arn:aws:iam::123456789012:role/my-role-1a",
    "my role 1b" = "arn:aws:iam::123456789012:role/my-role-1b",
    "my role 2"  = "arn:aws:iam::210987654321:role/my-role-2",
  }
  regions = [
    "us-east-1",
    "us-east-2",
    "us-west-1"
  ]
}
