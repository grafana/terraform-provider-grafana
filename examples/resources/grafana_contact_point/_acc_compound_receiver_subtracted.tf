resource "grafana_contact_point" "compound_contact_point" {
  name = "Compound Contact Point"

  email {
    disable_resolve_message = true
    addresses               = ["one@company.org", "two@company.org"]
  }
}
