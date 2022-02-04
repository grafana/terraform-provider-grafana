resource "grafana_api_key" "foo" {
  name = "key_foo"
  role = "Viewer"
}

resource "grafana_api_key" "bar" {
  name            = "key_bar"
  role            = "Admin"
  seconds_to_live = 30
}


output "api_key_foo_key_only" {
  value     = grafana_api_key.foo.key
  sensitive = true
}

output "api_key_bar" {
  value = grafana_api_key.bar
}
