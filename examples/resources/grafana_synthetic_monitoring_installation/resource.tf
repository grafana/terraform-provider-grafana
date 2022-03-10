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
  stack_id              = grafana_cloud_stack.sm_stack.id
  metrics_instance_id   = grafana_cloud_stack.sm_stack.prometheus_user_id
  logs_instance_id      = grafana_cloud_stack.sm_stack.logs_user_id
  metrics_publisher_key = grafana_cloud_api_key.metrics_publish.key
}
