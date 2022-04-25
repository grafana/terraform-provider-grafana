data "grafana_oncall_slack_channel" "example_slack_channel" {
  name = "example_slack_channel"
}

data "grafana_oncall_escalation_chain" "default" {
  name = "default"
}

resource "grafana_oncall_integration" "example_integration" {
  name = "Grafana Integration"
  type = "grafana"
}

resource "oncall_route" "example_route" {
  integration_id      = grafana_oncall_integration.example_integration.id
  escalation_chain_id = data.grafana_oncall_escalation_chain.default.id
  routing_regex       = "us-(east|west)"
  position            = 0
  slack {
    channel_id = data.grafana_oncall_slack_channel.example_slack_channel.slack_id
  }
}