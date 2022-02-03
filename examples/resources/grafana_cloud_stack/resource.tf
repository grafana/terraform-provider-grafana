resource "grafana_cloud_stack" "test" {
  name        = "gcloudstacktest"
  slug        = "gcloudstacktest"
  region_slug = "eu"
  description = "Test Grafana Cloud Stack"
}
