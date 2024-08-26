# __generated__ by Terraform
# Please review these resources and move them into your main configuration files.

# __generated__ by Terraform from "large-dashboard-test"
resource "grafana_dashboard" "large-dashboard-test" {
  config_json = file("${path.module}/dashboards/large-dashboard-test.json")
  folder      = grafana_folder.folder-with-large-dashboard.uid
}

# __generated__ by Terraform from "folder-with-large-dashboard"
resource "grafana_folder" "folder-with-large-dashboard" {
  title = "Folder with Large Dashboard"
  uid   = "folder-with-large-dashboard"
}
