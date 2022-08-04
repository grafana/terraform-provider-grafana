resource "grafana_contact_point" "compound_custom_contact_point" {
    name = "Compound Custom Contact Point"

    custom {
        type = "email"
        disable_resolve_message = true
        settings = {
            "addresses" = "one@company.org;two@company.org"
        }
    }
}
