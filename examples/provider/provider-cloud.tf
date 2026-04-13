variable "cloud_access_policy_token" {
  type        = string
  description = <<-EOT
    Cloud access policy token for Grafana Cloud with the following scopes:
    accesspolicies:read|write|delete, stacks:read|write|delete, stack-service-accounts:write
  EOT
}

// Step 1: Set up the cloud provider and create a stack
provider "grafana" {
  alias                     = "cloud"
  cloud_access_policy_token = var.cloud_access_policy_token
}

resource "grafana_cloud_stack" "stack" {
  provider = grafana.cloud

  name        = "myteststack"
  slug        = "myteststack"
  region_slug = "prod-us-east-0"
}

// Step 2: Create a stack service account for Grafana, OnCall, ML, SLO, and Asserts
resource "grafana_cloud_stack_service_account" "sa" {
  provider   = grafana.cloud
  stack_slug = grafana_cloud_stack.stack.slug

  name = "terraform-sa"
  role = "Admin"
}

resource "grafana_cloud_stack_service_account_token" "sa_token" {
  provider   = grafana.cloud
  stack_slug = grafana_cloud_stack.stack.slug

  name               = "terraform-sa-token"
  service_account_id = grafana_cloud_stack_service_account.sa.id
}

// Step 3: Create an access policy and token for Cloud Provider, Connections,
//         Fleet Management, and Frontend Observability
resource "grafana_cloud_access_policy" "all_services" {
  provider = grafana.cloud

  name   = "terraform-all-services"
  region = grafana_cloud_stack.stack.region_slug

  scopes = [
    // Cloud Provider (AWS/Azure) and Connections API (Metrics Endpoint scrape jobs)
    "integration-management:read",
    "integration-management:write",

    // Fleet Management
    "fleet-management:read",
    "fleet-management:write",

    // Frontend Observability
    "frontend-observability:read",
    "frontend-observability:write",
    "frontend-observability:delete",

    // Required by Cloud Provider, Connections, and Frontend Observability
    "stacks:read",
  ]

  realm {
    type       = "stack"
    identifier = grafana_cloud_stack.stack.id
  }
}

resource "grafana_cloud_access_policy_token" "all_services" {
  provider = grafana.cloud

  name             = "terraform-all-services-token"
  region           = grafana_cloud_access_policy.all_services.region
  access_policy_id = grafana_cloud_access_policy.all_services.policy_id
}

// Step 4: Configure a single provider for all services
//         All URLs are sourced from the grafana_cloud_stack resource.
provider "grafana" {
  // Grafana (dashboards, folders, alerting, users, etc.), OnCall, ML, SLO, Asserts
  url  = grafana_cloud_stack.stack.url
  auth = grafana_cloud_stack_service_account_token.sa_token.key

  // Cloud Provider (AWS/Azure)
  cloud_provider_url          = grafana_cloud_stack.stack.cloud_provider_url
  cloud_provider_access_token = grafana_cloud_access_policy_token.all_services.token

  // Connections API (Metrics Endpoint scrape jobs)
  connections_api_url          = grafana_cloud_stack.stack.connections_api_url
  connections_api_access_token = grafana_cloud_access_policy_token.all_services.token

  // Fleet Management
  fleet_management_url  = grafana_cloud_stack.stack.fleet_management_url
  fleet_management_auth = "${grafana_cloud_stack.stack.fleet_management_user_id}:${grafana_cloud_access_policy_token.all_services.token}"

  // Frontend Observability
  frontend_o11y_api_access_token = grafana_cloud_access_policy_token.all_services.token
}
