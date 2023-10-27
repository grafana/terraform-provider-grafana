variable "cloud_api_key" {
  description = "Cloud Access Policy token for Grafana Cloud with the following scopes: accesspolicies:read|write|delete, stacks:read|write|delete"
}
variable "stack_slug" {}
variable "cloud_region" {
  default = "us"
}

// Step 1: Create a stack
provider "grafana" {
  alias         = "cloud"
  cloud_api_key = var.cloud_api_key
}

resource "grafana_cloud_stack" "sm_stack" {
  provider = grafana.cloud

  name        = var.stack_slug
  slug        = var.stack_slug
  region_slug = var.cloud_region
}

// Step 2: Install Synthetic Monitoring on the stack
resource "grafana_cloud_access_policy" "sm_metrics_publish" {
  provider = grafana.cloud
  region   = var.cloud_region
  name     = "metric-publisher-for-sm"

  scopes = ["metrics:write", "stacks:read"]

  realm {
    type       = "stack"
    identifier = grafana_cloud_stack.sm_stack.id
  }
}

resource "grafana_cloud_access_policy_token" "sm_metrics_publish" {
  provider         = grafana.cloud
  region           = var.cloud_region
  access_policy_id = grafana_cloud_access_policy.sm_metrics_publish.policy_id
  name             = "metric-publisher-for-sm"
}

resource "grafana_synthetic_monitoring_installation" "sm_stack" {
  provider              = grafana.cloud
  stack_id              = grafana_cloud_stack.sm_stack.id
  metrics_publisher_key = grafana_cloud_access_policy_token.sm_metrics_publish.token
}


// Step 3: Interact with Synthetic Monitoring
provider "grafana" {
  alias           = "sm"
  sm_access_token = grafana_synthetic_monitoring_installation.sm_stack.sm_access_token
  sm_url          = grafana_synthetic_monitoring_installation.sm_stack.stack_sm_api_url
}

data "grafana_synthetic_monitoring_probes" "main" {
  provider   = grafana.sm
  depends_on = [grafana_synthetic_monitoring_installation.sm_stack]
}

resource "grafana_synthetic_monitoring_check" "ping" {
  provider = grafana.sm

  job     = "Ping Default"
  target  = "grafana.com"
  enabled = false
  probes = [
    data.grafana_synthetic_monitoring_probes.main.probes.Atlanta,
  ]
  labels = {
    foo = "bar"
  }
  settings {
    ping {}
  }
}
