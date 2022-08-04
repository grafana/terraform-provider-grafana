resource "grafana_contact_point" "my_contact_point" {
    name = "My Contact Point"

   email {
        disable_resolve_message = false
        addresses = ["one@company.org", "two@company.org"]
    }
}
