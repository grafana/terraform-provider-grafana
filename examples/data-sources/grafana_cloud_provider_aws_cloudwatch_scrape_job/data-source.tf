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
  stack_id                = data.grafana_cloud_stack.test.id
  name                    = "my-cloudwatch-scrape-job"
  aws_account_resource_id = grafana_cloud_provider_aws_account.test.resource_id
  export_tags             = true

  service {
    name = "AWS/EC2"
    metric {
      name       = "CPUUtilization"
      statistics = ["Average"]
    }
    metric {
      name       = "StatusCheckFailed"
      statistics = ["Maximum"]
    }
    scrape_interval_seconds = 300
    resource_discovery_tag_filter {
      key   = "k8s.io/cluster-autoscaler/enabled"
      value = "true"
    }
    tags_to_add_to_metrics = ["eks:cluster-name"]
  }

  custom_namespace {
    name = "CoolApp"
    metric {
      name       = "CoolMetric"
      statistics = ["Maximum", "Sum"]
    }
    scrape_interval_seconds = 300
  }

  static_label {
    label = "label1"
    value = "value1"
  }

  static_label {
    label = "label2"
    value = "value2"
  }
}


data "grafana_cloud_provider_aws_cloudwatch_scrape_job" "test" {
  stack_id = data.grafana_cloud_stack.test.id
  name     = grafana_cloud_provider_aws_cloudwatch_scrape_job.test.name
}
