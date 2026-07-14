---
layout: ""
page_title: "Provider: Grafana"
description: |-
  The Grafana provider provides configuration management resources for Grafana.
---

# Grafana Provider

The Grafana provider provides configuration management resources for
[Grafana](https://grafana.com/).

The changelog for this provider can be found here: <https://github.com/grafana/terraform-provider-grafana/releases>.

## Example Usage

### Creating a Grafana provider

```terraform
provider "grafana" {
  url  = "http://grafana.example.com/"
  auth = var.grafana_auth
}

// Optional (On-premise, not supported in Grafana Cloud): Create an organization
resource "grafana_organization" "my_org" {
  name = "my_org"
}

// Create resources (optional: within the organization)
resource "grafana_folder" "my_folder" {
  org_id = grafana_organization.my_org.org_id
  title  = "Test Folder"
}

resource "grafana_dashboard" "test_folder" {
  org_id = grafana_organization.my_org.org_id
  folder = grafana_folder.my_folder.id
  config_json = jsonencode({
    "title" : "My Dashboard Title",
    "uid" : "my-dashboard-uid"
    // ... other dashboard properties
  })
}
```

### Creating a Grafana Cloud stack provider

Before using the Terraform Provider to manage Grafana Cloud resources, you need to create an access policy token on the [Grafana Cloud Portal](https://grafana.com/docs/grafana-cloud/security-and-account-management/authentication-and-permissions/access-policies/create-access-policies/#create-access-policies-and-tokens). This initial token is used to create the stack, service accounts, and additional access policy tokens for the various Grafana Cloud services. The required scopes are `accesspolicies:read|write|delete`, `stacks:read|write|delete`, and `stack-service-accounts:write`.

```terraform
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
```

For Synthetic Monitoring and k6 setup, see the [`grafana_synthetic_monitoring_installation`](resources/synthetic_monitoring_installation.md) and [`grafana_k6_installation`](resources/k6_installation.md) resources, which include comprehensive examples.

<!-- schema generated by tfplugindocs -->
## Schema

### Optional

- `auth` (String, Sensitive) API token, basic auth in the `username:password` format or `anonymous` (string literal). May alternatively be set via the `GRAFANA_AUTH` environment variable.
- `ca_cert` (String) Certificate CA bundle (file path or literal value) to use to verify the Grafana server's certificate. May alternatively be set via the `GRAFANA_CA_CERT` environment variable.
- `cloud_access_policy_token` (String, Sensitive) Access Policy Token for Grafana Cloud. May alternatively be set via the `GRAFANA_CLOUD_ACCESS_POLICY_TOKEN` environment variable.
- `cloud_api_url` (String) Grafana Cloud's API URL. May alternatively be set via the `GRAFANA_CLOUD_API_URL` environment variable.
- `cloud_provider_access_token` (String, Sensitive) A Grafana Cloud Provider access token. May alternatively be set via the `GRAFANA_CLOUD_PROVIDER_ACCESS_TOKEN` environment variable.
- `cloud_provider_url` (String) A Grafana Cloud Provider backend address. May alternatively be set via the `GRAFANA_CLOUD_PROVIDER_URL` environment variable.
- `connections_api_access_token` (String, Sensitive) A Grafana Connections API access token. May alternatively be set via the `GRAFANA_CONNECTIONS_API_ACCESS_TOKEN` environment variable.
- `connections_api_url` (String) A Grafana Connections API address. May alternatively be set via the `GRAFANA_CONNECTIONS_API_URL` environment variable.
- `fleet_management_auth` (String, Sensitive) A Grafana Fleet Management basic auth in the `username:password` format. May alternatively be set via the `GRAFANA_FLEET_MANAGEMENT_AUTH` environment variable.
- `fleet_management_url` (String) A Grafana Fleet Management API address. May alternatively be set via the `GRAFANA_FLEET_MANAGEMENT_URL` environment variable.
- `frontend_o11y_api_access_token` (String, Sensitive) A Grafana Frontend Observability API access token. May alternatively be set via the `GRAFANA_FRONTEND_O11Y_API_ACCESS_TOKEN` environment variable.
- `frontend_o11y_api_url` (String) The Grafana Frontend Observability API URL. This is optional, and should only be set to override the default API. May alternatively be set via the `GRAFANA_FRONTEND_O11Y_API_URL` environment variable.
- `http_headers` (Map of String, Sensitive) Optional. HTTP headers mapping keys to values used for accessing the Grafana and Grafana Cloud APIs. May alternatively be set via the `GRAFANA_HTTP_HEADERS` environment variable in JSON format.
- `insecure_skip_verify` (Boolean) Skip TLS certificate verification. May alternatively be set via the `GRAFANA_INSECURE_SKIP_VERIFY` environment variable.
- `k6_access_token` (String, Sensitive) The k6 Cloud API token. May alternatively be set via the `GRAFANA_K6_ACCESS_TOKEN` environment variable.
- `k6_url` (String) The k6 Cloud API url. May alternatively be set via the `GRAFANA_K6_URL` environment variable.
- `oncall_access_token` (String, Sensitive) A Grafana OnCall access token. May alternatively be set via the `GRAFANA_ONCALL_ACCESS_TOKEN` environment variable. This is only required when using a dedicated OnCall API token. When using Grafana Cloud, OnCall can be accessed through the `auth` and `url` provider attributes instead.
- `oncall_url` (String) A Grafana OnCall backend address. May alternatively be set via the `GRAFANA_ONCALL_URL` environment variable. This is only required when using Grafana OnCall OSS. In Grafana Cloud, the OnCall URL is automatically inferred from the Grafana instance URL.
- `org_id` (Number) The Grafana org ID, if you are using a self-hosted OSS or enterprise Grafana instance. May alternatively be set via the `GRAFANA_ORG_ID` environment variable.
- `retries` (Number) The amount of retries to use for Grafana API and Grafana Cloud API calls. May alternatively be set via the `GRAFANA_RETRIES` environment variable.
- `retry_status_codes` (Set of String) The status codes to retry on for Grafana API and Grafana Cloud API calls. Use `x` as a digit wildcard. Defaults to 429 and 5xx. May alternatively be set via the `GRAFANA_RETRY_STATUS_CODES` environment variable.
- `retry_wait` (Number) The amount of time in seconds to wait between retries for Grafana API and Grafana Cloud API calls. May alternatively be set via the `GRAFANA_RETRY_WAIT` environment variable.
- `sm_access_token` (String, Sensitive) A Synthetic Monitoring access token. May alternatively be set via the `GRAFANA_SM_ACCESS_TOKEN` environment variable.
- `sm_url` (String) Synthetic monitoring backend address. May alternatively be set via the `GRAFANA_SM_URL` environment variable. The correct value for each service region is cited in the [Synthetic Monitoring documentation](https://grafana.com/docs/grafana-cloud/testing/synthetic-monitoring/set-up/set-up-private-probes/#probe-api-server-url). Note the `sm_url` value is optional, but it must correspond with the value specified as the `region_slug` in the `grafana_cloud_stack` resource. Also note that when a Terraform configuration contains multiple provider instances managing SM resources associated with the same Grafana stack, specifying an explicit `sm_url` set to the same value for each provider ensures all providers interact with the same SM API.
- `stack_id` (Number) The Grafana stack ID, if you are using a Grafana Cloud stack. May alternatively be set via the `GRAFANA_STACK_ID` environment variable.
- `store_dashboard_sha256` (Boolean) Set to true if you want to save only the sha256sum instead of complete dashboard model JSON in the tfstate.
- `tls_cert` (String) Client TLS certificate (file path or literal value) to use to authenticate to the Grafana server. May alternatively be set via the `GRAFANA_TLS_CERT` environment variable.
- `tls_key` (String) Client TLS key (file path or literal value) to use to authenticate to the Grafana server. May alternatively be set via the `GRAFANA_TLS_KEY` environment variable.
- `url` (String) The root URL of a Grafana server. May alternatively be set via the `GRAFANA_URL` environment variable.

## Authentication

One, or many, of the following authentication settings must be set. Each authentication setting allows a subset of resources to be used

### `auth`

This can be a Grafana API key, basic auth `username:password`, or a
[Grafana Service Account token](https://grafana.com/docs/grafana/latest/developers/http_api/examples/create-api-tokens-for-org/).

### `cloud_access_policy_token`

An access policy token created on the [Grafana Cloud Portal](https://grafana.com/docs/grafana-cloud/security-and-account-management/authentication-and-permissions/access-policies/using-an-access-policy-token/).

### `sm_access_token`

[Grafana Synthetic Monitoring](https://grafana.com/docs/grafana-cloud/testing/synthetic-monitoring/) uses distinct tokens for API access.
You can use the `grafana_synthetic_monitoring_installation` resource as shown above or you can request a new Synthetic Monitoring API key in Synthetics -> Config page.

### `oncall_access_token`

[Grafana OnCall](https://grafana.com/docs/oncall/latest/oncall-api-reference/)
uses API keys to allow access to the API. You can request a new OnCall API key in OnCall -> Settings page.

### `cloud_provider_access_token`

An access policy token created to manage [Grafana Cloud Provider Observability](https://grafana.com/docs/grafana-cloud/monitor-infrastructure/monitor-cloud-provider/).
To create one, follow the instructions in the [obtaining cloud provider access token section](#obtaining-cloud-provider-access-token).

### `connections_api_access_token`

An access policy token created on the [Grafana Cloud Portal](https://grafana.com/docs/grafana-cloud/security-and-account-management/authentication-and-permissions/access-policies/using-an-access-policy-token/) to manage
connections resources, such as Metrics Endpoint jobs.
For guidance on creating one, see section [obtaining connections access token](#obtaining-connections-access-token).

### `fleet_management_auth`

[Grafana Fleet Management](https://grafana.com/docs/grafana-cloud/send-data/fleet-management/api-reference/)
uses basic auth to allow access to the API, where the username is the Fleet Management instance ID and the
password is the API token. You can access the instance ID and request a new Fleet Management API token on the
Connections -> Collector -> Fleet Management page, in the API tab.

### `frontend_o11y_access_token`

An access policy token created on the [Grafana Cloud Portal](https://grafana.com/docs/grafana-cloud/security-and-account-management/authentication-and-permissions/access-policies/) to manage Frontend Observability apps.
For guidance on creating one, see section [obtaining Frontend Observability access token](#obtaining-frontend-observability-access-token).
