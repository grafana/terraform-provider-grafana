resource "grafana_contact_point" "my_contact_point" {
    name = "My Contact Point"

    custom {
        type = "email"
        disable_resolve_message = false
        settings = {
            "addresses" = "one@company.org;two@company.org"
        }
    }
}
