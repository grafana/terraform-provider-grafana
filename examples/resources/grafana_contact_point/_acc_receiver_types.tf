resource "grafana_contact_point" "receiver_types" {
    name = "Receiver Types"

    discord {
        url = "discord-url"
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
}
