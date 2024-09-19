# __generated__ by Terraform
# Please review these resources and move them into your main configuration files.

# __generated__ by Terraform from "email receiver"
resource "grafana_contact_point" "email_receiver" {
  disable_provenance = true
  name               = "email receiver"
  email {
    addresses               = ["<example@email.com>"]
    disable_resolve_message = false
    single_email            = false
  }
}

# __generated__ by Terraform from "my-dashboard-uid"
resource "grafana_dashboard" "my-dashboard-uid" {
  config_json = jsonencode({
    title = "My Dashboard"
    uid   = "my-dashboard-uid"
  })
  folder = grafana_folder.my-folder-uid.uid
}

# __generated__ by Terraform from "my-folder-uid"
resource "grafana_folder" "my-folder-uid" {
  title = "My Folder"
  uid   = "my-folder-uid"
}

# __generated__ by Terraform from "policy"
resource "grafana_notification_policy" "policy" {
  contact_point      = "grafana-default-email"
  disable_provenance = true
  group_by           = ["grafana_folder", "alertname"]
}

# __generated__ by Terraform from "1"
resource "grafana_organization_preferences" "_1" {
}

# __generated__ by Terraform
resource "grafana_user" "admin" {
  email    = "admin@localhost"
  is_admin = true
  login    = "admin"
  password = "SENSITIVE_VALUE_TO_REPLACE"
}
