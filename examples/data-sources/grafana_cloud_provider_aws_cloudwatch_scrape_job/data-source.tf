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

locals {
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
          key   = "k8s.io/cluster-autoscaler/enabled",
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
      is_custom_namespace     = true,
    },
  ]
}

resource "grafana_cloud_provider_aws_cloudwatch_scrape_job" "test" {
  stack_id                = data.grafana_cloud_stack.test.id
  name                    = "my-cloudwatch-scrape-job"
  aws_account_resource_id = grafana_cloud_provider_aws_account.test.resource_id
  regions                 = grafana_cloud_provider_aws_account.test.regions
  dynamic "service_configuration" {
    for_each = local.service_configurations
    content {
      name = service_configuration.value.name
      dynamic "metric" {
        for_each = service_configuration.value.metrics
        content {
          name       = metric.value.name
          statistics = metric.value.statistics
        }
      }
      scrape_interval_seconds = service_configuration.value.scrape_interval_seconds
      dynamic "resource_discovery_tag_filter" {
        for_each = lookup(service_configuration.value, "resource_discovery_tag_filters", [])
        content {
          key   = resource_discovery_tag_filter.value.key
          value = resource_discovery_tag_filter.value.value
        }

      }
      tags_to_add_to_metrics = lookup(service_configuration.value, "tags_to_add_to_metrics", [])
      is_custom_namespace    = lookup(service_configuration.value, "is_custom_namespace", false)
    }
  }
}


data "grafana_cloud_provider_aws_cloudwatch_scrape_job" "test" {
  stack_id = data.grafana_cloud_stack.test.id
  name     = grafana_cloud_provider_aws_cloudwatch_scrape_job.test.name
}
