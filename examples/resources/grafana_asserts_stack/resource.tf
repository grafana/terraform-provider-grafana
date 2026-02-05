# Example: Asserts Stack with Cloud Access Policy Token
#
# This example shows how to configure the Asserts stack using existing
# Terraform resources to create the required tokens.

# Step 1: Create a Cloud Access Policy with required scopes
resource "grafana_cloud_access_policy" "asserts" {
  name         = "asserts-stack-policy"
  display_name = "Asserts Stack Policy"

  scopes = [
    "stacks:read",   # For GCom API access
    "metrics:read",  # For Mimir metrics access
    "metrics:write", # For Mimir metrics access
  ]

  realm {
    type       = "stack"
    identifier = var.stack_id
  }
}

# Step 2: Create a token from the Cloud Access Policy
resource "grafana_cloud_access_policy_token" "asserts" {
  name             = "asserts-stack-token"
  access_policy_id = grafana_cloud_access_policy.asserts.policy_id
}

# Step 3: Create a Grafana Service Account for dashboards and Grafana Managed Alerts
# Required permissions: dashboards:create/write/read, folders:create/write/read/delete,
# datasources:read/query, alert.provisioning:write, alert.notifications.provisioning:write,
# alert.notifications:write, alert.rules:read/create/delete
resource "grafana_cloud_stack_service_account" "asserts" {
  stack_slug  = var.stack_slug
  name        = "asserts-managed-alerts-sa"
  role        = "Admin"
  is_disabled = false
}

resource "grafana_cloud_stack_service_account_token" "asserts" {
  stack_slug         = var.stack_slug
  service_account_id = grafana_cloud_stack_service_account.asserts.id
  name               = "asserts-managed-alerts-token"
}

# Step 4: Configure the Asserts Stack
resource "grafana_asserts_stack" "main" {
  # Required: Cloud Access Policy token for GCom, Mimir, and assertion detector
  cloud_access_policy_token = grafana_cloud_access_policy_token.asserts.token

  # Grafana Service Account token for dashboards and Grafana Managed Alerts
  grafana_token = grafana_cloud_stack_service_account_token.asserts.key
}

# Variables
variable "stack_id" {
  description = "The Grafana Cloud stack ID"
  type        = string
}

variable "stack_slug" {
  description = "The Grafana Cloud stack slug"
  type        = string
}

# Outputs
output "stack_enabled" {
  value       = grafana_asserts_stack.main.enabled
  description = "Whether the Asserts stack is enabled"
}

output "stack_status" {
  value       = grafana_asserts_stack.main.status
  description = "Current onboarding status of the stack"
}

output "stack_version" {
  value       = grafana_asserts_stack.main.version
  description = "Configuration version number"
}
