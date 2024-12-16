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

```terraform
// Step 1: Create a stack
provider "grafana" {
  alias                     = "cloud"
  cloud_access_policy_token = "my-token"
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
variable "cloud_access_policy_token" {
  description = "Cloud Access Policy token for Grafana Cloud with the following scopes: accesspolicies:read|write|delete, stacks:read|write|delete"
}
variable "stack_slug" {}
variable "cloud_region" {
  default = "us"
}

// Step 1: Create a stack
provider "grafana" {
  alias                     = "cloud"
  cloud_access_policy_token = var.cloud_access_policy_token
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
  scopes = ["metrics:write", "stacks:read", "logs:write", "traces:write"]
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

Alternatively, you can also configure the provider block by setting `url`
to your Grafana URL and `auth` to a service account token:

```terraform
// Step 1: Configure provider block.
provider "grafana" {
  alias = "oncall"
  url   = "http://grafana.example.com/"
  auth  = var.grafana_auth
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
- `cloud_access_policy_token` (String, Sensitive) Access Policy Token for Grafana Cloud. May alternatively be set via the `GRAFANA_CLOUD_ACCESS_POLICY_TOKEN` environment variable.
- `cloud_api_url` (String) Grafana Cloud's API URL. May alternatively be set via the `GRAFANA_CLOUD_API_URL` environment variable.
- `cloud_provider_access_token` (String, Sensitive) A Grafana Cloud Provider access token. May alternatively be set via the `GRAFANA_CLOUD_PROVIDER_ACCESS_TOKEN` environment variable.
- `cloud_provider_url` (String) A Grafana Cloud Provider backend address. May alternatively be set via the `GRAFANA_CLOUD_PROVIDER_URL` environment variable.
- `connections_api_access_token` (String, Sensitive) A Grafana Connections API access token. May alternatively be set via the `GRAFANA_CONNECTIONS_API_ACCESS_TOKEN` environment variable.
- `connections_api_url` (String) A Grafana Connections API address. May alternatively be set via the `GRAFANA_CONNECTIONS_API_URL` environment variable.
- `fleet_management_auth` (String, Sensitive) A Grafana Fleet Management basic auth in the `username:password` format. May alternatively be set via the `GRAFANA_FLEET_MANAGEMENT_AUTH` environment variable.
- `fleet_management_url` (String) A Grafana Fleet Management API address. May alternatively be set via the `GRAFANA_FLEET_MANAGEMENT_URL` environment variable.
- `http_headers` (Map of String, Sensitive) Optional. HTTP headers mapping keys to values used for accessing the Grafana and Grafana Cloud APIs. May alternatively be set via the `GRAFANA_HTTP_HEADERS` environment variable in JSON format.
- `insecure_skip_verify` (Boolean) Skip TLS certificate verification. May alternatively be set via the `GRAFANA_INSECURE_SKIP_VERIFY` environment variable.
- `oncall_access_token` (String, Sensitive) A Grafana OnCall access token. May alternatively be set via the `GRAFANA_ONCALL_ACCESS_TOKEN` environment variable.
- `oncall_url` (String) An Grafana OnCall backend address. May alternatively be set via the `GRAFANA_ONCALL_URL` environment variable.
- `retries` (Number) The amount of retries to use for Grafana API and Grafana Cloud API calls. May alternatively be set via the `GRAFANA_RETRIES` environment variable.
- `retry_status_codes` (Set of String) The status codes to retry on for Grafana API and Grafana Cloud API calls. Use `x` as a digit wildcard. Defaults to 429 and 5xx. May alternatively be set via the `GRAFANA_RETRY_STATUS_CODES` environment variable.
- `retry_wait` (Number) The amount of time in seconds to wait between retries for Grafana API and Grafana Cloud API calls. May alternatively be set via the `GRAFANA_RETRY_WAIT` environment variable.
- `sm_access_token` (String, Sensitive) A Synthetic Monitoring access token. May alternatively be set via the `GRAFANA_SM_ACCESS_TOKEN` environment variable.
- `sm_url` (String) Synthetic monitoring backend address. May alternatively be set via the `GRAFANA_SM_URL` environment variable. The correct value for each service region is cited in the [Synthetic Monitoring documentation](https://grafana.com/docs/grafana-cloud/testing/synthetic-monitoring/set-up/set-up-private-probes/#probe-api-server-url). Note the `sm_url` value is optional, but it must correspond with the value specified as the `region_slug` in the `grafana_cloud_stack` resource. Also note that when a Terraform configuration contains multiple provider instances managing SM resources associated with the same Grafana stack, specifying an explicit `sm_url` set to the same value for each provider ensures all providers interact with the same SM API.
- `store_dashboard_sha256` (Boolean) Set to true if you want to save only the sha256sum instead of complete dashboard model JSON in the tfstate.
- `tls_cert` (String) Client TLS certificate (file path or literal value) to use to authenticate to the Grafana server. May alternatively be set via the `GRAFANA_TLS_CERT` environment variable.
- `tls_key` (String) Client TLS key (file path or literal value) to use to authenticate to the Grafana server. May alternatively be set via the `GRAFANA_TLS_KEY` environment variable.
- `url` (String) The root URL of a Grafana server. May alternatively be set via the `GRAFANA_URL` environment variable.

### Managing Cloud Provider

#### Obtaining Cloud Provider access token

Before using the Terraform Provider to manage Grafana Cloud Provider Observability resources, such as AWS CloudWatch scrape jobs, you need to create an access policy token on the Grafana Cloud Portal. This token is used to authenticate the provider to the Grafana Cloud Provider API.
[These docs](https://grafana.com/docs/grafana-cloud/account-management/authentication-and-permissions/access-policies/authorize-services/#create-an-access-policy-for-a-stack) will guide you on how to create
an access policy. The required permissions, or scopes, are `integration-management:read`, `integration-management:write` and `stacks:read`.

Also, by default the Access Policies UI will not show those scopes, to find name you need to use the `Add Scope` textbox, as shown in the following image:

<img src="https://grafana.com/media/docs/grafana-cloud/aws/cloud-provider-terraform-access-policy-creation.png" width="700"/>

Having created an Access Policy, you can now create a token that will be used to authenticate the provider to the Cloud Provider API. You can do so just after creating the access policy, following
the in-screen instructions, of following [this guide](https://grafana.com/docs/grafana-cloud/account-management/authentication-and-permissions/access-policies/authorize-services/#create-one-or-more-access-policy-tokens).

#### Obtaining Cloud Provider API hostname

Having created the token, we can find the correct Cloud Provider API hostname by running the following script, that requires `curl` and [`jq`](https://jqlang.github.io/jq/) installed:

```bash
curl -sH "Authorization: Bearer <Access Token from previous step>" "https://grafana.com/api/instances" | \
  jq '[.items[]|{stackName: .slug, clusterName:.clusterSlug, cloudProviderAPIURL: "https://cloud-provider-api-\(.clusterSlug).grafana.net"}]'
```

This script will return a list of all the Grafana Cloud stacks you own, with the Cloud Provider API hostname for each one. Choose the correct hostname for the stack you want to manage.
For example, in the following response, the correct hostname for the `herokublogpost` stack is `https://cloud-provider-api-prod-us-central-0.grafana.net`.

```json
[
  {
    "stackName": "herokublogpost",
    "clusterName": "prod-us-central-0",
    "cloudProviderAPIURL": "https://cloud-provider-api-prod-us-central-0.grafana.net"
  }
]
```

#### Configuring the Provider to use the Cloud Provider API

Once you have the token and Cloud Provider API hostanme, you can configure the provider as follows:

```hcl
provider "grafana" {
  // ...
  cloud_provider_url = <Cloud Provider API URL from previous step>
  cloud_provider_access_token = <Access Token from previous step>
}
```

The following are examples on how the *Account* and *Scrape Job* resources can be configured:

```terraform
data "grafana_cloud_stack" "test" {
  slug = "gcloudstacktest"
}

data "aws_iam_role" "test" {
  name = "my-role"
}

resource "grafana_cloud_provider_aws_account" "test" {
  stack_id = data.grafana_cloud_stack.test.id
  role_arn = data.aws_iam_role.test.arn
  regions = [
    "us-east-1",
    "us-east-2",
    "us-west-1"
  ]
}
```

```terraform
data "grafana_cloud_stack" "test" {
  slug = "gcloudstacktest"
}

data "aws_iam_role" "test" {
  name = "my-role"
}

resource "grafana_cloud_provider_aws_account" "test" {
  stack_id = data.grafana_cloud_stack.test.id
  role_arn = data.aws_iam_role.test.arn
  regions = [
    "us-east-1",
    "us-east-2",
    "us-west-1"
  ]
}

resource "grafana_cloud_provider_aws_cloudwatch_scrape_job" "test" {
  stack_id                = data.grafana_cloud_stack.test.id
  name                    = "my-cloudwatch-scrape-job"
  aws_account_resource_id = grafana_cloud_provider_aws_account.test.resource_id
  export_tags             = true

  service {
    name = "AWS/EC2"
    metric {
      name       = "CPUUtilization"
      statistics = ["Average"]
    }
    metric {
      name       = "StatusCheckFailed"
      statistics = ["Maximum"]
    }
    scrape_interval_seconds = 300
    resource_discovery_tag_filter {
      key   = "k8s.io/cluster-autoscaler/enabled"
      value = "true"
    }
    tags_to_add_to_metrics = ["eks:cluster-name"]
  }

  custom_namespace {
    name = "CoolApp"
    metric {
      name       = "CoolMetric"
      statistics = ["Maximum", "Sum"]
    }
    scrape_interval_seconds = 300
  }
}
```

### Managing Connections

#### Obtaining Connections access token

Before using the Terraform Provider to manage Grafana Connections resources, such as metrics endpoint scrape jobs, you need to create an access policy token on the Grafana Cloud Portal. This token is used to authenticate the provider to the Grafana Connections API.
[These docs](https://grafana.com/docs/grafana-cloud/account-management/authentication-and-permissions/access-policies/authorize-services/#create-an-access-policy-for-a-stack) will guide you on how to create
an access policy. The required permissions, or scopes, are `integration-management:read`, `integration-management:write` and `stacks:read`.

Also, by default the Access Policies UI will not show those scopes, instead, search for it using the `Add Scope` textbox, as shown in the following image:

<img src="https://grafana.com/media/docs/grafana-cloud/connections/connections-terraform-access-policy-create.png" width="700"/>

1. Use the `Add Scope` textbox to search for the permissions you need to add to the access policy: `integration-management:read`, `integration-management:write` and `stacks:read`.
1. Once done, you should see the scopes selected with checkboxes.

Having created an Access Policy, you can now create a token that will be used to authenticate the provider to the Connections API. You can do so just after creating the access policy, following
the in-screen instructions, of following [this guide](https://grafana.com/docs/grafana-cloud/account-management/authentication-and-permissions/access-policies/authorize-services/#create-one-or-more-access-policy-tokens).

#### Obtaining Connections API hostname

Having created the token, we can find the correct Connections API hostname by running the following script, that requires `curl` and [`jq`](https://jqlang.github.io/jq/) installed:

```bash
curl -sH "Authorization: Bearer <Access Token from previous step>" "https://grafana.com/api/instances" | \
  jq '[.items[]|{stackName: .slug, clusterName:.clusterSlug, connectionsAPIURL: "https://connections-api-\(.clusterSlug).grafana.net"}]'
```

This script will return a list of all the Grafana Cloud stacks you own, with the Connections API hostname for each one. Choose the correct hostname for the stack you want to manage.
For example, in the following response, the correct hostname for the `examplestackname` stack is `https://connections-api-prod-eu-west-0.grafana.net`.

```json
[
  {
    "stackName": "examplestackname",
    "clusterName": "prod-eu-west-0",
    "connectionsAPIURL": "https://connections-api-prod-eu-west-0.grafana.net"
  }
]
```

#### Configuring the Provider to use the Connections API

Once you have the token and Connections API hostname, you can configure the provider as follows:

```hcl
provider "grafana" {
  connections_api_url          = "<Connections API URL from previous step>"
  connections_api_access_token = "<Access Token from previous step>"
}
```

### Managing Grafana Fleet Management

```terraform
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
```

## Authentication

One, or many, of the following authentication settings must be set. Each authentication setting allows a subset of resources to be used

### `auth`

This can be a Grafana API key, basic auth `username:password`, or a
[Grafana Service Account token](https://grafana.com/docs/grafana/latest/developers/http_api/examples/create-api-tokens-for-org/).

### `cloud_access_policy_token`

An access policy token created on the [Grafana Cloud Portal](https://grafana.com/docs/grafana-cloud/account-management/authentication-and-permissions/access-policies/authorize-services/).

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

An access policy token created on the [Grafana Cloud Portal](https://grafana.com/docs/grafana-cloud/account-management/authentication-and-permissions/access-policies/authorize-services/) to manage
connections resources, such as Metrics Endpoint jobs.
For guidance on creating one, see section [obtaining connections access token](#obtaining-connections-access-token).

### `fleet_management_auth`

[Grafana Fleet Management](https://grafana.com/docs/grafana-cloud/send-data/fleet-management/api-reference/)
uses basic auth to allow access to the API, where the username is the Fleet Management instance ID and the
password is the API key. You can access the instance ID and request a new Fleet Management API key on the
Connections -> Collector -> Fleet Management page, in the API tab.
