resource "grafana_contact_point" "receiver_types" {
  name = "Receiver Types since v11.3"

  mqtt {
    broker_url     = "tcp://localhost:1883"
    client_id      = "grafana"
    topic          = "grafana/alerts"
    message_format = "json"
    username       = "user"
    password       = "password123"
    qos            = 1
    retain         = true
    tls_config {
      insecure_skip_verify = true
      ca_certificate       = "ca_cert"
      client_certificate   = "client_cert"
      client_key           = "client_key"
    }
  }
}
