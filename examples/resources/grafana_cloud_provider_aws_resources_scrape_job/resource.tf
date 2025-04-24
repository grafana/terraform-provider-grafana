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

resource "grafana_cloud_provider_aws_resources_scrape_job" "test" {
  stack_id                = data.grafana_cloud_stack.test.id
  name                    = "my-aws-resources-scrape-job"
  aws_account_resource_id = grafana_cloud_provider_aws_account.test.resource_id

  service {
    name = "AWS/EC2"
    scrape_interval_seconds = 300
    resource_discovery_tag_filter {
      key   = "k8s.io/cluster-autoscaler/enabled"
      value = "true"
    }
  }

  static_labels = {
    "label1" = "value1"
    "label2" = "value2"
  }
}
