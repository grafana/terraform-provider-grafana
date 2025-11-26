resource "grafana_cloud_stack_service_account" "cloud_sa" {
  stack_slug = "<your stack slug>"

  name        = "cloud service account"
  role        = "Admin"
  is_disabled = false
}

resource "grafana_cloud_stack_service_account_rotating_token" "foo" {
  stack_slug = "<your stack slug>"

  name_prefix                   = "key_foo"
  service_account_id            = grafana_cloud_stack_service_account.cloud_sa.id
  seconds_to_live               = 7776000 # 3 months
  early_rotation_window_seconds = 604800  # 1 week
}

output "service_account_token_foo_key" {
  value     = grafana_cloud_stack_service_account_token.foo.key
  sensitive = true
}
