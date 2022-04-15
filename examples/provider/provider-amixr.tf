// Step 1: Create a stack it you haven't already.
provider "grafana" {
  auth = "cloud_api_key"
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
  auth               = "amixr_access_token"
  alias              = "amixr"
  amixr_access_token = "my_amixr_token"
}

data "grafana_amixr_user" "alex" {
  username = "alex"
}

// Step 3: Interact with Grafana OnCall
resource "grafana_amixr_integration" "test-acc-integration" {
  provider = grafana.amixr
  name     = "my integration"
  type     = "grafana"
  default_route {
    escalation_chain_id = grafana_amixr_escalation_chain.default.id
  }
}

resource "grafana_amixr_escalation_chain" "default" {
  provider = grafana.amixr
  name     = "default"
}

resource "grafana_amixr_escalation" "example_notify_step" {
  escalation_chain_id = grafana_amixr_escalation_chain.default.id
  type = "notify_persons"
  persons_to_notify = [
    data.grafana_amixr_user.alex.id
  ]
  position = 0
}

