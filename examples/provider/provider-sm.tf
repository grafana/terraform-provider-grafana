// Step 1: Create a stack
provider "grafana" {
  alias         = "cloud"
  cloud_api_key = "<my-api-key>"
  sm_url        = "<synthetic-monitoring-api-url>"
}

resource "grafana_cloud_stack" "sm_stack" {
  provider = grafana.cloud

  name        = "<stack-name>"
  slug        = "<stack-slug>"
  region_slug = "us"
}

// Step 2: Install Synthetic Monitoring on the stack
resource "grafana_cloud_api_key" "metrics_publish" {
  provider = grafana.cloud

  name           = "MetricsPublisherForSM"
  role           = "MetricsPublisher"
  cloud_org_slug = "<org-slug>"
}

resource "grafana_synthetic_monitoring_installation" "sm_stack" {
  provider = grafana.cloud

  stack_id              = grafana_cloud_stack.sm_stack.id
  metrics_instance_id   = grafana_cloud_stack.sm_stack.prometheus_user_id
  logs_instance_id      = grafana_cloud_stack.sm_stack.logs_user_id
  metrics_publisher_key = grafana_cloud_api_key.metrics_publish.key
}

// Step 3: Interact with Synthetic Monitoring
provider "grafana" {
  alias           = "sm"
  sm_access_token = grafana_synthetic_monitoring_installation.sm_stack.sm_access_token
  sm_url          = "<synthetic-monitoring-api-url>"
}

data "grafana_synthetic_monitoring_probes" "main" {
  provider = grafana.sm
  depends_on = [
    grafana_synthetic_monitoring_installation.sm_stack
  ]
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
