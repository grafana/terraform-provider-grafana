data "grafana_cloud_observability_aws_accounts" "test" {
  stack_id       = grafana_cloud_stack.test.id
  name           = "my-aws-connection"
  aws_account_id = "1234567"
  regions        = ["us-east-1", "us-east-2", "us-west-1"]
  role_arn       = "my role"
}
