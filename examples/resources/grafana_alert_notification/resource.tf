resource "grafana_alert_notification" "email_someteam" {
  name          = "Email that team"
  type          = "email"
  is_default    = false
  send_reminder = true
  frequency     = "24h"

  settings = {
    addresses   = "foo@example.net;bar@example.net"
    uploadImage = "false"
  }
}
