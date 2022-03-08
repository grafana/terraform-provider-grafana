resource "grafana_cloud_plugin_installation" "test" {
  stack_slug = "stackname"
  slug       = "some-plugin"
  version    = "1.2.3"
}