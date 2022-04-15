resource "grafana_amixr_escalation_chain" "default" {
  provider = grafana.amixr
  name     = "default"
}

data "grafana_amixr_user" "alex" {
  username = "alex"
}

// Notify step
resource "grafana_amixr_escalation" "example_notify_step" {
  escalation_chain_id = grafana_amixr_escalation_chain.default.id
  type = "notify_persons"
  persons_to_notify = [
    data.grafana_amixr_user.alex.id
  ]
  position = 0
}

// Wait step
resource "grafana_amixr_escalation" "example_notify_step" {
  escalation_chain_id = grafana_amixr_escalation_chain.default.id
  type = "wait"
  duration = 300
  position = 1
}

// Important step
resource "grafana_amixr_escalation" "example_notify_step" {
  escalation_chain_id = grafana_amixr_escalation_chain.default.id
  type = "notify_persons"
  important = true
  persons_to_notify = [
    data.grafana_amixr_user.alex.id
  ]
  position = 0
}
