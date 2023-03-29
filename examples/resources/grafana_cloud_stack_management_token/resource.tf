resource "grafana_cloud_stack_management_token" "management_token" {
  stack_slug = "<your stack slug>"

  name        = "management_token"
  role        = "Admin"
  is_disabled = false

}

output "management_token" {
  value     = grafana_cloud_stack_management_token.management_token.token
  sensitive = true
}
