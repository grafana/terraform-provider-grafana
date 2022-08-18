resource "grafana_contact_point" "receiver_types" {
    name = "Receiver Types"

    alertmanager {
        url = "http://my-am"
        basic_auth_user = "user"
        basic_auth_password = "password"
    }

    dingding {
        url = "http://dingding-url"
        message_type = "link"
        message = "message"
    }

    discord {
        url = "http://discord-url"
        message = "message"
        avatar_url = "avatar_url"
        use_discord_username = true
        disable_resolve_message = true
    }

    email {
        addresses = ["one@company.org", "two@company.org"]
        message = "message"
        subject = "subject"
        single_email = true
        disable_resolve_message = true
    }

    googlechat {
        url = "http://googlechat-url"
        message = "message"
    }

    kafka {
        rest_proxy_url = "http://kafka-rest-proxy-url"
        topic = "mytopic"
    }

    opsgenie {
        url = "http://opsgenie-api"
        api_key = "token"
        message = "message"
        description = "description"
        auto_close = true
        override_priority = true
        send_tags_as = "both"
    }

    pagerduty {
        integration_key = "token"
        severity = "critical"
        class = "ping failure"
        component = "mysql"
        group = "my service"
        summary = "message"
    }

    pushover {
        user_key = "userkey"
        api_token = "token"
        priority = 0
        ok_priority = 0
        retry = 45
        expire = 80000
        device = "device"
        sound = "bugle"
        ok_sound = "cashregister"
        message = "message"
    }

    sensugo {
        url = "http://sensugo-url"
        api_key = "key"
        entity = "entity"
        check = "check"
        namespace = "namespace"
        handler = "handler"
        message = "message"
    }

    slack {
        endpoint_url = "http://custom-slack-url"
        token = "xoxb-token"
        recipient = "#channel"
        text = "message"
        title = "title"
        username = "bot"
        icon_emoji = ":icon:"
        icon_url = "http://domain/icon.png"
        mention_channel = "here"
        mention_users = "user"
        mention_groups = "group"
    }

    teams {
        url = "http://teams-webhook"
        message = "message"
        title = "title"
        section_title = "section"
    }

    telegram {
        token = "token"
        chat_id = "chat-id"
        message = "message"
    }

    threema {
        gateway_id = "*gateway"
        recipient_id = "*target1"
        api_secret = "secret"
    }
}
