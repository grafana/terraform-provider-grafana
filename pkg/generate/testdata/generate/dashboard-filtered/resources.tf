# __generated__ by Terraform
# Please review these resources and move them into your main configuration files.

# __generated__ by Terraform from "my-dashboard-uid"
resource "grafana_dashboard" "my-dashboard-uid" {
  config_json = jsonencode({
    title = "My Dashboard"
    uid   = "my-dashboard-uid"
  })
  folder = "my-folder-uid"
}
