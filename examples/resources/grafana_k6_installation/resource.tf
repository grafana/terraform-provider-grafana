variable "cloud_access_policy_token" {
  description = "Cloud Access Policy token for Grafana Cloud with the following scopes: stacks:read|write|delete, stack-service-accounts:write"
}
variable "stack_slug" {}
variable "cloud_region" {
  default = "us"
}

// Step 1: Create a stack in Grafana Cloud
provider "grafana" {
  alias                     = "cloud"
  cloud_access_policy_token = var.cloud_access_policy_token
}

resource "grafana_cloud_stack" "k6_stack" {
  provider = grafana.cloud

  name        = var.stack_slug
  slug        = var.stack_slug
  region_slug = var.cloud_region
}

// Step 2: Create a Service Account and a token to install the k6 App
resource "grafana_cloud_stack_service_account" "k6_sa" {
  provider   = grafana.cloud
  stack_slug = var.stack_slug

  name        = "${var.stack_slug}-k6-app"
  role        = "Admin"
  is_disabled = false
}

resource "grafana_cloud_stack_service_account_token" "k6_sa_token" {
  provider   = grafana.cloud
  stack_slug = var.stack_slug

  name               = "${var.stack_slug}-k6-app-token"
  service_account_id = grafana_cloud_stack_service_account.k6_sa.id
}

// Step 3: Install the k6 App on the stack
resource "grafana_k6_installation" "k6_installation" {
  provider = grafana.cloud

  cloud_access_policy_token = var.cloud_access_policy_token
  stack_id                  = grafana_cloud_stack.k6_stack.id
  grafana_sa_token          = grafana_cloud_stack_service_account_token.k6_sa_token.key
  grafana_user              = "admin"
}

// Step 4: Interact with the k6 App: create a new project
provider "grafana" {
  alias = "k6"

  stack_id        = grafana_cloud_stack.k6_stack.id
  k6_access_token = grafana_k6_installation.k6_installation.k6_access_token
}

resource "grafana_k6_project" "my_k6_project" {
  provider = grafana.k6

  name = "k6 Project created with TF"
}
