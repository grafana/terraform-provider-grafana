resource "grafana_contact_point" "receiver_types" {
  name = "Receiver Types since v9.3"

  victorops {
    url          = "http://victor-ops-url"
    message_type = "CRITICAL"
    title        = "title"
    description  = "description"
  }

}
