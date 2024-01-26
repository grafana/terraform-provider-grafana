---
layout: ""
page_title: "Provider: Grafana"
description: |-
  The Grafana provider provides configuration management resources for Grafana.
---

# Grafana Provider

The Grafana provider provides configuration management resources for
[Grafana](https://grafana.com/).

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

```terraform
// Step 1: Create a stack
provider "grafana" {
  alias         = "cloud"
  cloud_api_key = "my-token"
}

resource "grafana_cloud_stack" "my_stack" {
  provider = grafana.cloud

  name        = "myteststack"
  slug        = "myteststack"
  region_slug = "us"
}

// Step 2: Create a service account and key for the stack
resource "grafana_cloud_stack_service_account" "cloud_sa" {
  provider   = grafana.cloud
  stack_slug = grafana_cloud_stack.my_stack.slug

  name        = "cloud service account"
  role        = "Admin"
  is_disabled = false
}

resource "grafana_cloud_stack_service_account_token" "cloud_sa" {
  provider   = grafana.cloud
  stack_slug = grafana_cloud_stack.my_stack.slug

  name               = "my_stack cloud_sa key"
  service_account_id = grafana_cloud_stack_service_account.cloud_sa.id
}

// Step 3: Create resources within the stack
provider "grafana" {
  alias = "my_stack"

  url  = grafana_cloud_stack.my_stack.url
  auth = grafana_cloud_stack_service_account_token.cloud_sa.key
}

resource "grafana_folder" "my_folder" {
  provider = grafana.my_stack

  title = "Test Folder"
}
```

### Installing Synthetic Monitoring on a new Grafana Cloud Stack

```terraform
variable "cloud_api_key" {
  description = "Cloud Access Policy token for Grafana Cloud with the following scopes: accesspolicies:read|write|delete, stacks:read|write|delete"
}
variable "stack_slug" {}
variable "cloud_region" {
  default = "us"
}

// Step 1: Create a stack
provider "grafana" {
  alias         = "cloud"
  cloud_api_key = var.cloud_api_key
}

resource "grafana_cloud_stack" "sm_stack" {
  provider = grafana.cloud

  name        = var.stack_slug
  slug        = var.stack_slug
  region_slug = var.cloud_region
}

// Step 2: Install Synthetic Monitoring on the stack
resource "grafana_cloud_access_policy" "sm_metrics_publish" {
  provider = grafana.cloud

  region = var.cloud_region
  name   = "metric-publisher-for-sm"
  scopes = ["metrics:write", "stacks:read"]
  realm {
    type       = "stack"
    identifier = grafana_cloud_stack.sm_stack.id
  }
}

resource "grafana_cloud_access_policy_token" "sm_metrics_publish" {
  provider = grafana.cloud

  region           = var.cloud_region
  access_policy_id = grafana_cloud_access_policy.sm_metrics_publish.policy_id
  name             = "metric-publisher-for-sm"
}

resource "grafana_synthetic_monitoring_installation" "sm_stack" {
  provider = grafana.cloud

  stack_id              = grafana_cloud_stack.sm_stack.id
  metrics_publisher_key = grafana_cloud_access_policy_token.sm_metrics_publish.token
}


// Step 3: Interact with Synthetic Monitoring
provider "grafana" {
  alias           = "sm"
  sm_access_token = grafana_synthetic_monitoring_installation.sm_stack.sm_access_token
  sm_url          = grafana_synthetic_monitoring_installation.sm_stack.stack_sm_api_url
}

data "grafana_synthetic_monitoring_probes" "main" {
  provider   = grafana.sm
  depends_on = [grafana_synthetic_monitoring_installation.sm_stack]
}
```

### Managing Grafana OnCall

```terraform
// Step 1: Configure provider block.
// Go to the Grafana OnCall in your stack and create api token in the settings tab.It will be your oncall_access_token.
// If you are using Grafana OnCall OSS consider set oncall_url. You can get it in OnCall -> settings -> API URL.
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
```

<!-- schema generated by tfplugindocs -->
## Schema

### Optional

- `auth` (String, Sensitive) API token, basic auth in the `username:password` format or `anonymous` (string literal). May alternatively be set via the `GRAFANA_AUTH` environment variable.
- `ca_cert` (String) Certificate CA bundle (file path or literal value) to use to verify the Grafana server's certificate. May alternatively be set via the `GRAFANA_CA_CERT` environment variable.
- `cloud_api_key` (String, Sensitive) Access Policy Token (or API key) for Grafana Cloud. May alternatively be set via the `GRAFANA_CLOUD_API_KEY` environment variable.
- `cloud_api_url` (String) Grafana Cloud's API URL. May alternatively be set via the `GRAFANA_CLOUD_API_URL` environment variable.
- `http_headers` (Map of String, Sensitive) Optional. HTTP headers mapping keys to values used for accessing the Grafana and Grafana Cloud APIs. May alternatively be set via the `GRAFANA_HTTP_HEADERS` environment variable in JSON format.
- `insecure_skip_verify` (Boolean) Skip TLS certificate verification. May alternatively be set via the `GRAFANA_INSECURE_SKIP_VERIFY` environment variable.
- `oncall_access_token` (String, Sensitive) A Grafana OnCall access token. May alternatively be set via the `GRAFANA_ONCALL_ACCESS_TOKEN` environment variable.
- `oncall_url` (String) An Grafana OnCall backend address. May alternatively be set via the `GRAFANA_ONCALL_URL` environment variable.
- `org_id` (Number, Deprecated) Deprecated: Use the `org_id` attributes on resources instead.
- `retries` (Number) The amount of retries to use for Grafana API and Grafana Cloud API calls. May alternatively be set via the `GRAFANA_RETRIES` environment variable.
- `retry_status_codes` (Set of String) The status codes to retry on for Grafana API and Grafana Cloud API calls. Use `x` as a digit wildcard. Defaults to 429 and 5xx. May alternatively be set via the `GRAFANA_RETRY_STATUS_CODES` environment variable.
- `retry_wait` (Number) The amount of time in seconds to wait between retries for Grafana API and Grafana Cloud API calls. May alternatively be set via the `GRAFANA_RETRY_WAIT` environment variable.
- `sm_access_token` (String, Sensitive) A Synthetic Monitoring access token. May alternatively be set via the `GRAFANA_SM_ACCESS_TOKEN` environment variable.
- `sm_url` (String) Synthetic monitoring backend address. May alternatively be set via the `GRAFANA_SM_URL` environment variable. The correct value for each service region is cited in the [Synthetic Monitoring documentation](https://grafana.com/docs/grafana-cloud/monitor-public-endpoints/private-probes/#probe-api-server-url). Note the `sm_url` value is optional, but it must correspond with the value specified as the `region_slug` in the `grafana_cloud_stack` resource. Also note that when a Terraform configuration contains multiple provider instances managing SM resources associated with the same Grafana stack, specifying an explicit `sm_url` set to the same value for each provider ensures all providers interact with the same SM API.
- `store_dashboard_sha256` (Boolean) Set to true if you want to save only the sha256sum instead of complete dashboard model JSON in the tfstate.
- `tls_cert` (String) Client TLS certificate (file path or literal value) to use to authenticate to the Grafana server. May alternatively be set via the `GRAFANA_TLS_CERT` environment variable.
- `tls_key` (String) Client TLS key (file path or literal value) to use to authenticate to the Grafana server. May alternatively be set via the `GRAFANA_TLS_KEY` environment variable.
- `url` (String) The root URL of a Grafana server. May alternatively be set via the `GRAFANA_URL` environment variable.

## Authentication

One, or many, of the following authentication settings must be set. Each authentication setting allows a subset of resources to be used

### `auth`

This can be a Grafana API key, basic auth `username:password`, or a
[Grafana API key](https://grafana.com/docs/grafana/latest/developers/http_api/create-api-tokens-for-org/).

### `cloud_api_key`

An access policy token created on the [Grafana Cloud Portal](https://grafana.com/docs/grafana-cloud/account-management/authentication-and-permissions/create-api-key/).

### `sm_access_token`

[Synthetic Monitoring](https://grafana.com/docs/grafana-cloud/monitor-public-endpoints/)
endpoints require a dedicated access token. You may obtain an access token with its
[Registration API](https://github.com/grafana/synthetic-monitoring-api-go-client/blob/main/docs/API.md#registration-api).

```console
curl \
  -X POST \
  -H 'Content-type: application/json; charset=utf-8' \
  -H "Authorization: Bearer $GRAFANA_CLOUD_API_KEY" \
  -d '{"stackId": <stack-id>, "metricsInstanceId": <metrics-instance-id>, "logsInstanceId": <logs-instance-id>}' \
  "$SM_API_URL/api/v1/register/install"
```

`GRAFANA_CLOUD_API_KEY` is an API key created on the
[Grafana Cloud Portal](https://grafana.com/docs/grafana-cloud/account-management/authentication-and-permissions/create-api-key/).
It must have the `MetricsPublisher` role.

`SM_API_URL` is the URL of the Synthetic Monitoring API.
Based on the region of your Grafana Cloud stack, you need to use a different API URL.

Please [see API docs](https://github.com/grafana/synthetic-monitoring-api-go-client/blob/main/docs/API.md#api-url) to find `SM_API_URL` for your region.

`stackId`, `metricsInstanceId`, and `logsInstanceId` may also be obtained on
the portal. First, you need to create a Stack by clicking "Add Stack". When it's
created you will be taken to its landing page on the portal. Get your `stackId`
from the URL in your browser:

```
https://grafana.com/orgs/<org-slug>/stacks/<stack-id>
```

Next, go to "Details" for Prometheus. Again, get `metricsInstanceId` from your URL:

```
https://grafana.com/orgs/<org-slug>/hosted-metrics/<metrics-instance-id>
```

Finally, go back to your stack page, and go to "Details" for Loki to get
`logsInstanceId`.

```
https://grafana.com/orgs/<org-slug>/hosted-logs/<logs-instance-id>
```

### `oncall_access_token`

[Grafana OnCall](https://grafana.com/docs/oncall/latest/oncall-api-reference/)
uses API keys to allow access to the API. You can request a new OnCall API key in OnCall -> Settings page.
