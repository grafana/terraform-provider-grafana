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
}
