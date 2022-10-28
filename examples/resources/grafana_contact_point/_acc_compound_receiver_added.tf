resource "grafana_contact_point" "compound_contact_point" {
  name = "Compound Contact Point"

  email {
    disable_resolve_message = true
    addresses               = ["one@company.org", "two@company.org"]
  }

  email {
    disable_resolve_message = true
    addresses               = ["three@company.org", "four@company.org"]
  }

  email {
    disable_resolve_message = true
    addresses               = ["five@company.org", "six@company.org"]
  }
}
