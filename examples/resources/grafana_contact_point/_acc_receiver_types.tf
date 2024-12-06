resource "grafana_contact_point" "receiver_types" {
  name = "Receiver Types"

  alertmanager {
    url                 = "http://my-am"
    basic_auth_user     = "user"
    basic_auth_password = "password"
  }

  dingding {
    url          = "http://dingding-url"
    message_type = "link"
    message      = "message"
    title        = "title"
  }

  discord {
    url                     = "http://discord-url"
    title                   = "title"
    message                 = "message"
    avatar_url              = "avatar_url"
    use_discord_username    = true
    disable_resolve_message = true
  }

  email {
    addresses               = ["one@company.org", "two@company.org"]
    message                 = "message"
    subject                 = "subject"
    single_email            = true
    disable_resolve_message = true
  }

  googlechat {
    url     = "http://googlechat-url"
    title   = "title"
    message = "message"
  }

  kafka {
    rest_proxy_url = "http://kafka-rest-proxy-url"
    topic          = "mytopic"
    description    = "description"
    details        = "details"
    username       = "username"
    password       = "password"
    api_version    = "v3"
    cluster_id     = "cluster_id"
  }

  line {
    token       = "token"
    title       = "title"
    description = "description"
  }

  mqtt {
    broker_url     = "tcp://localhost:1883"
    client_id      = "client_id"
    topic          = "grafana/alerts"
    message_format = "json"
    username       = "username"
    password       = "password"
    qos            = 1
    retain         = true
    tls_config {
      insecure_skip_verify = true
      ca_certificate       = "ca_cert"
      client_certificate   = "client_cert"
      client_key           = "client"
    }
  }

  opsgenie {
    url               = "http://opsgenie-api"
    api_key           = "token"
    message           = "message"
    description       = "description"
    auto_close        = true
    override_priority = true
    send_tags_as      = "both"
  }

  pagerduty {
    integration_key = "token"
    severity        = "critical"
    class           = "ping failure"
    component       = "mysql"
    group           = "my service"
    summary         = "message"
    source          = "source"
    client          = "client"
    client_url      = "http://pagerduty"
    details = {
      "one"   = "two"
      "three" = "four"
    }
    url = "http://pagerduty-url"
  }

  pushover {
    user_key     = "userkey"
    api_token    = "token"
    priority     = 0
    ok_priority  = 0
    retry        = 45
    expire       = 80000
    device       = "device"
    sound        = "bugle"
    ok_sound     = "cashregister"
    title        = "title"
    message      = "message"
    upload_image = false
  }

  sensugo {
    url       = "http://sensugo-url"
    api_key   = "key"
    entity    = "entity"
    check     = "check"
    namespace = "namespace"
    handler   = "handler"
    message   = "message"
  }

  slack {
    endpoint_url    = "http://custom-slack-url"
    token           = "xoxb-token"
    recipient       = "#channel"
    text            = "message"
    title           = "title"
    username        = "bot"
    icon_emoji      = ":icon:"
    icon_url        = "http://domain/icon.png"
    mention_channel = "here"
    mention_users   = "user"
    mention_groups  = "group"
  }

  teams {
    url           = "http://teams-webhook"
    message       = "message"
    title         = "title"
    section_title = "section"
  }

  telegram {
    token                    = "token"
    chat_id                  = "chat-id"
    message_thread_id        = "5"
    message                  = "message"
    parse_mode               = "Markdown"
    disable_web_page_preview = true
    protect_content          = true
    disable_notifications    = true
  }

  threema {
    gateway_id   = "*gateway"
    recipient_id = "*target1"
    api_secret   = "secret"
    title        = "title"
    description  = "description"
  }

  victorops {
    url          = "http://victor-ops-url"
    message_type = "CRITICAL"
    title        = "title"
    description  = "description"
  }

  webex {
    token   = "token"
    api_url = "http://localhost"
    message = "message"
    room_id = "room_id"
  }

  webhook {
    url                 = "http://my-url"
    http_method         = "POST"
    basic_auth_user     = "user"
    basic_auth_password = "password"
    max_alerts          = 100
    message             = "Custom message"
    title               = "Custom title"
  }

  wecom {
    url      = "http://wecom-url"
    message  = "message"
    title    = "title"
    secret   = "secret"
    corp_id  = "corp_id"
    agent_id = "agent_id"
    msg_type = "text"
    to_user  = "to_user"
  }
}
