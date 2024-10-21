resource "grafana_role" "report_admin_role" {
  name                   = "Report Administrator"
  uid                    = "report_admin_role_uid"
  auto_increment_version = true
  permissions {
    action = "reports:create"
  }
  permissions {
    action = "reports:read"
    scope  = "reports:*"
  }
}

resource "grafana_group_attribute_mapping" "report_admin_mapping" {
  group_id  = "business_dev_group_id"
  role_uids = [grafana_role.report_admin_role.uid]
}
