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
  services = [
    {
      name = "AWS/EC2",
      metrics = [
        {
          name = "CPUUtilization",
          statistics = [
            "Average",
          ],
        },
        {
          name = "StatusCheckFailed",
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
  ]
  custom_namespaces = [
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
    },
  ]
}

resource "grafana_cloud_provider_aws_cloudwatch_scrape_job" "test" {
  stack_id                = data.grafana_cloud_stack.test.id
  name                    = "my-cloudwatch-scrape-job"
  aws_account_resource_id = grafana_cloud_provider_aws_account.test.resource_id
  regions                 = grafana_cloud_provider_aws_account.test.regions
  export_tags             = true

  dynamic "service" {
    for_each = local.services
    content {
      name = service.value.name
      dynamic "metric" {
        for_each = service.value.metrics
        content {
          name       = metric.value.name
          statistics = metric.value.statistics
        }
      }
      scrape_interval_seconds = service.value.scrape_interval_seconds
      dynamic "resource_discovery_tag_filter" {
        for_each = service.value.resource_discovery_tag_filters
        content {
          key   = resource_discovery_tag_filter.value.key
          value = resource_discovery_tag_filter.value.value
        }

      }
      tags_to_add_to_metrics = service.value.tags_to_add_to_metrics
    }
  }

  dynamic "custom_namespace" {
    for_each = local.custom_namespaces
    content {
      name = custom_namespace.value.name
      dynamic "metric" {
        for_each = custom_namespace.value.metrics
        content {
          name       = metric.value.name
          statistics = metric.value.statistics
        }
      }
      scrape_interval_seconds = custom_namespace.value.scrape_interval_seconds
    }
  }
}
