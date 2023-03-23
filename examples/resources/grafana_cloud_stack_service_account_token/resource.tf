resource "grafana_cloud_stack_service_account" "cloud_sa" {
  stack_slug = "<your stack slug>"

  name        = "cloud service account"
  role        = "Admin"
  is_disabled = false
}

resource "grafana_service_account_token" "foo" {
  name               = "key_foo"
  service_account_id = grafana_cloud_stack_service_account.cloud_sa.id
}

output "service_account_token_foo_key_only" {
  value     = grafana_service_account_token.foo.key
  sensitive = true
}
