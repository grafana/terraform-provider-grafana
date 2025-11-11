resource "grafana_service_account" "test" {
  name = "test-service-account"
  role = "Viewer"
}

resource "grafana_service_account_rotating_token" "foo" {
  name_prefix                   = "key_foo"
  service_account_id            = grafana_service_account.test.id
  seconds_to_live               = 7776000 # 3 months
  early_rotation_window_seconds = 604800  # 1 week
}

output "service_account_token_foo_key" {
  value     = grafana_service_account_rotating_token.foo.key
  sensitive = true
}
