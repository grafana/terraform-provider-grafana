data "grafana_team" "my_team" {
  name = "my team"
}

data "grafana_oncall_team" "my_team" {
  name = data.grafana_team.my_team.name
}

resource "grafana_oncall_outgoing_webhook" "test-acc-outgoing_webhook" {
  provider = grafana.oncall
  name     = "my outgoing webhook"
  url      = "https://example.com/"

  // Optional: specify the team to which the outgoing webhook belongs
  team_id = data.grafana_oncall_team.my_team.id
}

resource "grafana_oncall_outgoing_webhook" "test-acc-outgoing_webhook-incident" {
  provider     = grafana.oncall
  name         = "my outgoing incident webhook"
  preset       = "incident_webhook"
  http_method  = "POST"
  url          = "https://example.com/"
  trigger_type = "incident declared"
}
