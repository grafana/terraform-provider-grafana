resource "grafana_scim_config" "default" {
  enable_user_sync  = true
  enable_group_sync = false
}
