resource "grafana_role" "super_user" {
  name        = "Super User"
  description = "My Super User description"
  uid         = "superuseruid"
  version     = 1
  global      = true

  permissions {
    action = "org.users:add"
    scope  = "users:*"
  }
  permissions {
    action = "org.users:write"
    scope  = "users:*"
  }
  permissions {
    action = "org.users:read"
    scope  = "users:*"
  }
}
