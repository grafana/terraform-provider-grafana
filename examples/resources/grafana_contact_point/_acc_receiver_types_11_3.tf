resource "grafana_contact_point" "receiver_types" {
  name = "Receiver Types since v11.3"

  # Basic MQTT configuration
  mqtt {
    broker_url = "tcp://localhost:1883"
    topic      = "grafana/alerts"
    qos        = 0
  }

  # MQTT with authentication
  mqtt {
    broker_url     = "tcp://localhost:1883"
    topic          = "grafana/alerts"
    client_id      = "grafana-client"
    username       = "mqtt-user"
    password       = "secret123"
    message_format = "json"
    qos            = 1
  }

  # MQTT with TLS
  mqtt {
    broker_url     = "ssl://localhost:8883"
    topic          = "grafana/alerts"
    client_id      = "grafana-secure"
    message_format = "json"
    qos            = 2
    retain         = true

    tls_config {
      insecure_skip_verify = false
      ca_certificate       = "ca cert"
      client_certificate   = "client cert"
      client_key           = "client key"
    }
  }
}
