resource "grafana_contact_point" "receiver_types" {
  name = "Receiver Types since v10.3"

  opsgenie {
    url               = "http://opsgenie-api"
    api_key           = "token"
    message           = "message"
    description       = "description"
    auto_close        = true
    override_priority = true
    send_tags_as      = "both"
    responders {
      type = "user"
      id   = "803f87e1a7f848b0a0779810bee5d1d3"
    }
    responders {
      type = "team"
      name = "Test team"
    }
  }
}
