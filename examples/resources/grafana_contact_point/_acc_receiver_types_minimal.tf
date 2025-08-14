resource "grafana_contact_point" "minimal_receivers" {
  name = "Minimal Receivers"

  alertmanager {
    url = "http://my-am"
  }

  dingding {
    url = "http://dingding-url"
  }

  discord {
    url = "http://discord-url"
  }

  email {
    addresses = ["one@company.org", "two@company.org"]
  }

  googlechat {
    url = "http://googlechat-url"
  }

  kafka {
    rest_proxy_url = "http://kafka-rest-proxy-url"
    topic          = "mytopic"
  }

  line {
    token = "token"
  }

  oncall {
    url = "http://oncall-url"
  }

  opsgenie {
    api_key = "token"
  }

  pagerduty {
    integration_key = "token"
  }

  pushover {
    user_key  = "userkey"
    api_token = "token"
  }

  sensugo {
    url     = "http://sensugo-url"
    api_key = "key"
  }

  slack {
    token     = "xoxb-token"
    recipient = "#channel"
  }

  slack {
    url = "http://custom-slack-url"
  }

  teams {
    url = "http://teams-webhook"
  }

  telegram {
    token   = "token"
    chat_id = "chat-id"
  }

  threema {
    gateway_id   = "*gateway"
    recipient_id = "*target1"
    api_secret   = "secret"
  }

  victorops {
    url = "http://victor-ops-url"
  }

  webex {
    token   = "token"
    room_id = "room_id"
  }

  webhook {
    url = "http://webhook-url"
  }

  wecom {
    url = "http://wecom-url"
  }

  wecom {
    secret   = "secret"
    corp_id  = "corp_id"
    agent_id = "agent_id"
  }
}
