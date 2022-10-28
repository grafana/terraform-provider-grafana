resource "grafana_contact_point" "default_settings" {
  name = "Default Settings"

  slack {
    token     = "xoxb-token"
    recipient = "#channel"
  }
}
