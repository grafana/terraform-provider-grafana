resource "grafana_dashboard" "localhost_1_my-dashboard-uid" {
  provider    = grafana.localhost
  config_json = file("${path.module}/files/localhost_1_my-dashboard-uid.json")
  folder      = "my-folder-uid"
  org_id      = jsonencode(1)
}

resource "grafana_notification_policy" "localhost_1_policy" {
  provider           = grafana.localhost
  contact_point      = "grafana-default-email"
  disable_provenance = true
  group_by           = ["grafana_folder", "alertname"]
  org_id             = jsonencode(1)
}

resource "grafana_folder" "localhost_1_my-folder-uid" {
  provider = grafana.localhost
  org_id   = jsonencode(1)
  title    = "My Folder"
  uid      = "my-folder-uid"
}
