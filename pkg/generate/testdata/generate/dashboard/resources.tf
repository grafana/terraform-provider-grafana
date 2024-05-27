# __generated__ by Terraform
# Please review these resources and move them into your main configuration files.

# __generated__ by Terraform from "1:my-dashboard-uid"
resource "grafana_dashboard" "_1_my-dashboard-uid" {
  config_json = jsonencode({
    title = "My Dashboard"
    uid   = "my-dashboard-uid"
  })
  folder = grafana_folder._1_my-folder-uid.uid
}

# __generated__ by Terraform from "1:my-folder-uid"
resource "grafana_folder" "_1_my-folder-uid" {
  title = "My Folder"
  uid   = "my-folder-uid"
}

# __generated__ by Terraform from "1:policy"
resource "grafana_notification_policy" "_1_policy" {
  contact_point      = "grafana-default-email"
  disable_provenance = true
  group_by           = ["grafana_folder", "alertname"]
}

# __generated__ by Terraform from "1"
resource "grafana_organization_preferences" "_1" {
}
