---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "grafana_cloud_private_datasource_connect_token Resource - terraform-provider-grafana"
subcategory: "Cloud"
description: |-
  Official documentation https://grafana.com/docs/grafana-cloud/connect-externally-hosted/private-data-source-connect/API documentation https://grafana.com/docs/grafana-cloud/developer-resources/api-reference/cloud-api/#create-a-token
  Required access policy scopes:
  accesspolicies:readaccesspolicies:writeaccesspolicies:delete
---

# grafana_cloud_private_datasource_connect_token (Resource)

* [Official documentation](https://grafana.com/docs/grafana-cloud/connect-externally-hosted/private-data-source-connect/)
* [API documentation](https://grafana.com/docs/grafana-cloud/developer-resources/api-reference/cloud-api/#create-a-token)

Required access policy scopes:

* accesspolicies:read
* accesspolicies:write
* accesspolicies:delete

## Example Usage

```terraform
data "grafana_cloud_stack" "current" {
  stackID = "<your stack ID>"
}

resource "grafana_cloud_private_datasource_connect" "test" {
  region       = "us"
  name         = "my-pdc"
  display_name = "My PDC"
  identifier   = data.grafana_cloud_stack.current.stackID
}

resource "grafana_cloud_private_datasource_connect_token" "test" {
  pdc_network_id = grafana_cloud_private_datasource_connect.test.network_id
  region         = "us"
  name           = "my-pdc-token"
  display_name   = "My PDC Token"
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `name` (String) Name of the private datasource network token.
- `pdc_network_id` (String) ID of the private datasource network for which to create a token.
- `region` (String) Region of the private datasource network. Should be set to the same region as the private datasource network. Use the region list API to get the list of available regions: https://grafana.com/docs/grafana-cloud/developer-resources/api-reference/cloud-api/#list-regions.

### Optional

- `display_name` (String) Display name of the private datasource network token. Defaults to the name.
- `expires_at` (String) Expiration date of the private datasource network token. Does not expire by default.

### Read-Only

- `created_at` (String) Creation date of the private datasource network token.
- `id` (String) The ID of this resource.
- `token` (String, Sensitive)
- `updated_at` (String) Last update date of the private datasource network token.

## Import

Import is supported using the following syntax:

```shell
terraform import grafana_cloud_private_datasource_connect_token.name "{{ region }}:{{ tokenId }}"
```