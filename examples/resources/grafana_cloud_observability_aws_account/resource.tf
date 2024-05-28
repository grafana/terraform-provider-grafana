provider "grafana" {
  alias = "cloud"
}

resource "grafana_cloud_stack" "test" {
  name        = "gcloudstacktest"
  slug        = "gcloudstacktest"
  region_slug = "eu"
  description = "Test Grafana Cloud Stack"
}

resource "grafana_cloud_observability_aws_account" "my-aws-account" {
  stack_id       = grafana_cloud_stack.test.id
  name           = "my-aws-connection"
  aws_account_id = "1234567"
  regions        = ["us-east-1", "us-east-2", "us-west-1"]
  role_arn       = "my role"
}
