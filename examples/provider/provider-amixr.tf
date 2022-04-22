// Step 1: Create a stack it you haven't already.
provider "grafana" {
  auth          = "cloud_api_key"
  alias         = "cloud"
  cloud_api_key = "<my-api-key>"
}

resource "grafana_cloud_stack" "sm_stack" {
  provider = grafana.cloud

  name        = "<stack-name>"
  slug        = "<stack-slug>"
  region_slug = "us"
}

// Step 2: Go to the Grafana OnCall in your stack and create api token in the settings tab.
provider "grafana" {
  auth                = "oncall_access_token"
  alias               = "oncall"
  oncall_access_token = "my_oncall_token"
}

data "grafana_oncall_user" "alex" {
  username = "alex"
}

// Step 3: Interact with Grafana OnCall
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

