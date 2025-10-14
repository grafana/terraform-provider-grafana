# __generated__ by Terraform
# Please review these resources and move them into your main configuration files.

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
