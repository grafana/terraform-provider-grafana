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

# The OnCall integration will provide a URL to use in the contact point
resource "grafana_oncall_integration" "grafana_cloud" {
  name = "Grafana Cloud Alerts"
  type = "grafana_alerting"
  default_route {
    escalation_chain_id = "..."
  }
}

resource "grafana_contact_point" "grafana_cloud" {
  name = "Grafana Cloud OnCall"
  oncall {
    url = grafana_oncall_integration.grafana_cloud.link
  }
}
