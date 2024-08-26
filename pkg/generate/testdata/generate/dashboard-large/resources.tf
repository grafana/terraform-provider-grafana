# __generated__ by Terraform
# Please review these resources and move them into your main configuration files.

# __generated__ by Terraform from "1:large-dashboard-test"
resource "grafana_dashboard" "_1_large-dashboard-test" {
  config_json = file("${path.module}/dashboards/_1_large-dashboard-test.json")
  folder      = grafana_folder._1_folder-with-large-dashboard.uid
}

# __generated__ by Terraform from "1:folder-with-large-dashboard"
resource "grafana_folder" "_1_folder-with-large-dashboard" {
  title = "Folder with Large Dashboard"
  uid   = "folder-with-large-dashboard"
}
