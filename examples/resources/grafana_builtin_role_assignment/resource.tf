resource "grafana_builtin_role_assignment" "viewer" {
  builtin_role = "Viewer"
  roles {
    uid    = "firstuid"
    global = false
  }
  roles {
    uid    = "seconduid"
    global = true
  }
}
