resource "grafana_role" "test" {
  name        = "test-role"
  description = "test-role description"
  uid         = "test-ds-role-uid"
  version     = 1
  global      = true
  hidden      = false

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

data "grafana_role" "from_name" {
  name = grafana_role.test.name
}
