data "grafana_cloud_stack" "teststack" {
  provider = grafana.cloud
  name     = "gcloudstacktest"
}

data "grafana_frontend_o11y_app" "test-app" {
  provider = grafana.cloud
  stack_id = data.grafana_cloud_stack.teststack.id
  name     = "test-app"
}
