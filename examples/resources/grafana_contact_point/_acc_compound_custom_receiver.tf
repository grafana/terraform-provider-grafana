resource "grafana_contact_point" "compound_custom_contact_point" {
    name = "Compound Custom Contact Point"

    custom {
        type = "email"
        disable_resolve_message = true
        settings = {
            "addresses" = "one@company.org;two@company.org"
        }
    }

    custom {
        type = "discord"
        disable_resolve_message = true
        settings = {
            "url" = "http://discord-webhook-url"
        }
    }
}