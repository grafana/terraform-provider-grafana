resource "grafana_service_account" "test" {
  name = "test-service-account"
  role = "Viewer"
}

resource "grafana_service_account_token" "foo" {
  name               = "key_foo"
  service_account_id = grafana_service_account.test.id
}

resource "grafana_service_account_token" "bar" {
  name               = "key_bar"
  service_account_id = grafana_service_account.test.id
  seconds_to_live    = 30
}


output "service_account_token_foo_key_only" {
  value     = grafana_service_account_token.foo.key
  sensitive = true
}

output "service_account_token_bar" {
  value     = grafana_service_account_token.bar
  sensitive = true
}
