resource "grafana_cloud_stack" "test" {
  name   = "grafanacloudstack-test"
  slug   = "grafanacloudstack-test"
  region_slug = "eu"
  description = "Test Grafana Cloud Stack"
}

data "grafana_cloud_stack" "test" {
  slug = grafana_cloud_stack.test.slug
}
