// Variables
variable "cloud_access_policy_token" {
  type        = string
  description = "Cloud access policy token with scopes: accesspolicies:read|write|delete, stacks:read|write|delete"
}

variable "stack_slug" {
  type        = string
  description = "Subdomain that the Grafana Cloud instance will be available at: https://<stack_slug>.grafana.net"
}

variable "region_slug" {
  type        = string
  description = "Region to assign to the stack"
  default     = "us"
}

// Step 1: Create a stack
provider "grafana" {
  alias = "cloud"

  cloud_access_policy_token = var.cloud_access_policy_token
}

resource "grafana_cloud_stack" "stack" {
  provider = grafana.cloud

  name        = var.stack_slug
  slug        = var.stack_slug
  region_slug = var.region_slug
}

// Step 2: Create an access policy and token for Fleet Management
resource "grafana_cloud_access_policy" "policy" {
  provider = grafana.cloud

  name   = "fleet-management-policy"
  region = grafana_cloud_stack.stack.region_slug

  scopes = [
    "fleet-management:read",
    "fleet-management:write"
  ]

  realm {
    type       = "stack"
    identifier = grafana_cloud_stack.stack.id
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

  fleet_management_auth = "${grafana_cloud_stack.stack.fleet_management_user_id}:${grafana_cloud_access_policy_token.token.token}"
  fleet_management_url  = grafana_cloud_stack.stack.fleet_management_url
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
    "collector.os=~\".*\"",
    "env=\"PROD\""
  ]
  enabled = true
}
