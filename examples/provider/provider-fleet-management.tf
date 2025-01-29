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

// Step 2: Create an access policy and token for Fleet Management
resource "grafana_cloud_access_policy" "policy" {
  provider = grafana.cloud

  name   = "fleet-management-policy"
  region = data.grafana_cloud_stack.stack.region_slug

  scopes = [
    "fleet-management:read",
    "fleet-management:write"
  ]

  realm {
    type       = "stack"
    identifier = data.grafana_cloud_stack.stack.id
  }
}

resource "grafana_cloud_access_policy_token" "token" {
  provider = grafana.cloud

  name             = "fleet-management-token"
  region           = grafana_cloud_access_policy.policy.region
  access_policy_id = grafana_cloud_access_policy.policy.policy_id
}

// Step 3: Interact with Fleet Management
provider "grafana" {
  alias = "fm"

  fleet_management_auth = "${data.grafana_cloud_stack.stack.fleet_management_user_id}:${grafana_cloud_access_policy_token.token.token}"
  fleet_management_url  = data.grafana_cloud_stack.stack.fleet_management_url
}

resource "grafana_fleet_management_collector" "collector" {
  provider = grafana.fm

  id = "my_collector"
  attribute_overrides = {
    "env"   = "PROD",
    "owner" = "TEAM-A"
  }
  enabled = true
}

resource "grafana_fleet_management_pipeline" "pipeline" {
  provider = grafana.fm

  name     = "my_pipeline"
  contents = file("config.alloy")
  matchers = [
    "collector.os=\"linux\"",
    "env=\"PROD\""
  ]
  enabled = true
}
