resource "grafana_role" "super_user" {
  name = "Super User"
  description = "My Super User description"
  uid = "superuseruid"
  version = 1
  global = true

  permissions {
    action = "users:create"
  }
  permissions {
    action = "users:read"
    scope = "global:users:*"
  }
}
