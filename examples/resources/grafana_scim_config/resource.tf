resource "grafana_scim_config" "default" {
  enable_user_sync            = true
  enable_group_sync           = false
  allow_non_provisioned_users = false
}
