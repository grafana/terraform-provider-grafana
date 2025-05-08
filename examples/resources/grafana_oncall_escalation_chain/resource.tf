data "grafana_team" "my_team" {
  name = "my team"
}

data "grafana_oncall_team" "my_team" {
  name = data.grafana_team.my_team.name
}

resource "grafana_oncall_escalation_chain" "default" {
  provider = grafana.oncall
  name     = "default"

  // Optional: specify the team to which the escalation chain belongs
  team_id = data.grafana_oncall_team.my_team.id
}
