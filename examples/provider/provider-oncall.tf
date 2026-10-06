// Step 1: Configure provider block with an OnCall API token, created in the IRM settings of your stack.
// Without url and auth, the OnCall API URL cannot be derived: set oncall_url to the API URL shown in the IRM settings
// unless your stack uses the default us-central OnCall backend.
provider "grafana" {
  alias               = "oncall"
  oncall_access_token = "my_oncall_token"
}

data "grafana_oncall_user" "alex" {
  username = "alex"
}

// Step 2: Interact with Grafana OnCall
resource "grafana_oncall_integration" "test-acc-integration" {
  provider = grafana.oncall
  name     = "my integration"
  type     = "grafana"
  default_route {
    escalation_chain_id = grafana_oncall_escalation_chain.default.id
  }
}

resource "grafana_oncall_escalation_chain" "default" {
  provider = grafana.oncall
  name     = "default"
}

resource "grafana_oncall_escalation" "example_notify_step" {
  escalation_chain_id = grafana_oncall_escalation_chain.default.id
  type                = "notify_persons"
  persons_to_notify = [
    data.grafana_oncall_user.alex.id
  ]
  position = 0
}

