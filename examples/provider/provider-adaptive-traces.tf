// Variables
variable "cloud_access_policy_token" {
  type        = string
  description = "Cloud access policy token with scopes: accesspolicies:read|write|delete, stacks:read"
}

variable "stack_slug" {
  type        = string
  description = "Subdomain that the Grafana Cloud instance is available at: https://<stack_slug>.grafana.net"
}

// Step 1: Retrieve stack details
provider "grafana" {
  alias = "cloud"

  cloud_access_policy_token = var.cloud_access_policy_token
}

data "grafana_cloud_stack" "stack" {
  provider = grafana.cloud

  slug = var.stack_slug
}

// Step 2: Create an access policy and token for Adaptive Traces
resource "grafana_cloud_access_policy" "policy" {
  provider = grafana.cloud

  name   = "adaptive-traces-policy"
  region = data.grafana_cloud_stack.stack.region_slug

  scopes = [
    "adaptive-traces-policies:read",
    "adaptive-traces-policies:write"
  ]

  realm {
    type       = "stack"
    identifier = data.grafana_cloud_stack.stack.id
  }
}

resource "grafana_cloud_access_policy_token" "token" {
  provider = grafana.cloud

  name             = "adaptive-traces-token"
  region           = grafana_cloud_access_policy.policy.region
  access_policy_id = grafana_cloud_access_policy.policy.policy_id
}

// Step 3: Manage Adaptive Traces sampling policies
provider "grafana" {
  alias = "traces"

  adaptive_traces_auth = "${data.grafana_cloud_stack.stack.traces_user_id}:${grafana_cloud_access_policy_token.token.token}"
  adaptive_traces_url  = data.grafana_cloud_stack.stack.traces_url
}

// Creating the first policy on a stack starts Adaptive Traces filtering.
resource "grafana_adaptive_traces_policy" "errors" {
  provider = grafana.traces

  name = "Keep all traces with errors"
  type = "status_code"
  body = jsonencode({
    status_codes = ["ERROR"]
  })
}
