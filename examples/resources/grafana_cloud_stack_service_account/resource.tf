resource "grafana_cloud_stack_service_account" "cloud_sa" {
  stack_slug = "<your stack slug>"

  name        = "cloud service account"
  role        = "Admin"
  is_disabled = false
}
