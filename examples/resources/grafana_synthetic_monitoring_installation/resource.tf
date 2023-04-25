resource "grafana_cloud_stack" "sm_stack" {
  name        = "<stack-name>"
  slug        = "<stack-slug>"
  region_slug = "us"
}

resource "grafana_cloud_api_key" "metrics_publish" {
  name           = "MetricsPublisherForSM"
  role           = "MetricsPublisher"
  cloud_org_slug = "<org-slug>"
}

resource "grafana_synthetic_monitoring_installation" "sm_stack" {
  stack_id = grafana_cloud_stack.sm_stack.id
}

// Create a new provider instance to interact with Synthetic Monitoring
provider "grafana" {
  alias           = "sm"
  sm_access_token = grafana_synthetic_monitoring_installation.sm_stack.sm_access_token
  sm_url          = "grafana_synthetic_monitoring_installation.sm_stack.stack_sm_api_url"
}
