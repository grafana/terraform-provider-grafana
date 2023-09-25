resource "grafana_contact_point" "my_contact_point" {
  name = "My Contact Point"

  email {
    addresses               = ["one@company.org", "two@company.org"]
    message                 = "{{ len .Alerts.Firing }} firing."
    subject                 = "{{ template \"default.title\" .}}"
    single_email            = true
    disable_resolve_message = false
  }
}
