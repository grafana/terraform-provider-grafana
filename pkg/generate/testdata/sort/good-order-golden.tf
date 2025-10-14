# __generated__ by Terraform
# Please review these resources and move them into your main configuration files.

# __generated__ by Terraform from "1:my-dashboard-uid"
resource "grafana_dashboard" "localhost_1_my-dashboard-uid" {
  provider    = grafana.localhost
  config_json = file("${path.module}/files/localhost_1_my-dashboard-uid.json")
  folder      = "my-folder-uid"
  org_id      = jsonencode(1)
}

# __generated__ by Terraform from "1:my-folder-uid"
resource "grafana_folder" "localhost_1_my-folder-uid" {
  provider = grafana.localhost
  org_id   = jsonencode(1)
  title    = "My Folder"
  uid      = "my-folder-uid"
}

# __generated__ by Terraform from "1:policy"
resource "grafana_notification_policy" "localhost_1_policy" {
  provider           = grafana.localhost
  contact_point      = "grafana-default-email"
  disable_provenance = true
  group_by           = ["grafana_folder", "alertname"]
  org_id             = jsonencode(1)
}
