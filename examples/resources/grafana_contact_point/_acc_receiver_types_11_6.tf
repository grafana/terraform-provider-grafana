resource "grafana_contact_point" "receiver_types" {
  name = "Receiver Types since v11.6"

  webhook {
    url = "http://hmac-minimal-webhook-url"
    hmac_config {
      secret = "test-hmac-minimal-secret"
    }
  }

  webhook {
    url = "http://hmac-webhook-url"
    hmac_config {
      secret           = "test-hmac-secret"
      header           = "X-Grafana-Alerting-Signature"
      timestamp_header = "X-Grafana-Alerting-Timestamp"
    }
  }
}
