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

resource "grafana_cloud_provider_aws_cloudwatch_scrape_job" "test" {
  stack_id = data.grafana_cloud_stack.test.id
  name = "my-cloudwatch-scrape-job"
  aws_account_resource_id = grafana_cloud_provider_aws_account.test.resource_id
  regions = grafana_cloud_provider_aws_account.regions
  service_configurations = [
    {
      name = "AWS/EC2",
      metrics = [
        {
          name = "aws_ec2_cpuutilization",
          statistics = [
            "Average",
          ],
        },
        {
          name = "aws_ec2_status_check_failed",
          statistics = [
            "Maximum",
          ],
        },
      ],
      scrape_interval_seconds = 300,
      resource_discovery_tag_filters = [
        {
          key = "k8s.io/cluster-autoscaler/enabled",
          value = "true",
        }
      ],
      tags_to_add_to_metrics = [
        "eks:cluster-name",
      ]
    },
    {
      name = "CoolApp",
      metrics = [
        {
          name = "CoolMetric",
          statistics = [
            "Maximum",
            "Sum",
          ]
        },
      ],
      scrape_interval_seconds = 300,
      is_custom_namespace = true,
    },
  ]
}
