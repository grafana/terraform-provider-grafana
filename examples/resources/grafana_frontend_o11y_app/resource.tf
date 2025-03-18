data "grafana_cloud_stack" "teststack" {
  provider = grafana.cloud
  name     = "gcloudstacktest"
}

resource "grafana_frontend_o11y_app" "test-app" {
  provider        = grafana.cloud
  stack_id        = data.grafana_cloud_stack.teststack.id
  name            = "test-app"
  allowed_origins = ["https://grafana.com"]

  extra_log_attributes = {
    "terraform" : "true"
  }

  settings = {
    "combineLabData" : "1"
  }
}
