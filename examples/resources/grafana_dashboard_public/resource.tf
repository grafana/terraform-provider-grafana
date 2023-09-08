resource "grafana_dashboard" "my_dashboard" {
  config_json = jsonencode({
    "title" : "My Terraform Dashboard",
    "uid" : "my-dashboard-uid"
  })
}
resource "grafana_dashboard_public" "my_public_dashboard" {
  dashboard_uid = grafana_dashboard.my_dashboard.uid

  uid          = "my-custom-public-uid"
  access_token = "e99e4275da6f410d83760eefa934d8d2"

  time_selection_enabled = true
  is_enabled             = true
  annotations_enabled    = true
  share                  = "public"
}
