resource "grafana_cloud_api_key" "test" {
  cloud_org_slug = "myorg"
  name           = "my-key"
  role           = "Admin"
}
