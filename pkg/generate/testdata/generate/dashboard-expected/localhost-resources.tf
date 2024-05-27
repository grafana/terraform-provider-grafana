# __generated__ by Terraform
# Please review these resources and move them into your main configuration files.

# __generated__ by Terraform from "1:my-dashboard-uid"
resource "grafana_dashboard" "localhost_1_my-dashboard-uid" {
  provider = grafana.localhost
  config_json = jsonencode({
    title = "My Dashboard"
    uid   = "my-dashboard-uid"
  })
  folder = "my-folder-uid"
}

# __generated__ by Terraform from "1:my-folder-uid"
resource "grafana_folder" "localhost_1_my-folder-uid" {
  provider = grafana.localhost
  title    = "My Folder"
  uid      = "my-folder-uid"
}

# __generated__ by Terraform from "1:policy"
resource "grafana_notification_policy" "localhost_1_policy" {
  provider           = grafana.localhost
  contact_point      = "grafana-default-email"
  disable_provenance = true
  group_by           = ["grafana_folder", "alertname"]
}
