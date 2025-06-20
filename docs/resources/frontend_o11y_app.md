---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "grafana_frontend_o11y_app Resource - terraform-provider-grafana"
subcategory: "Frontend Observability"
description: |-
  
---

# grafana_frontend_o11y_app (Resource)



## Example Usage

```terraform
data "grafana_cloud_stack" "teststack" {
  provider = grafana.cloud
  name     = "gcloudstacktest"
}

resource "grafana_frontend_o11y_app" "test-app" {
  provider        = grafana.cloud
  stack_id        = data.grafana_cloud_stack.teststack.id
  name            = "test-app"
  allowed_origins = ["https://grafana.com"]

  extra_log_attributes = {
    "terraform" : "true"
  }

  settings = {
    "combineLabData"               = "1",
    "geolocation.enabled"          = "1",
    "geolocation.level"            = "3",
    "geolocation.country_denylist" = "DE,GR"
  }
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `allowed_origins` (List of String) A list of allowed origins for CORS.
- `extra_log_attributes` (Map of String) The extra attributes to append in each signal.
- `name` (String) The name of Frontend Observability App. Part of the Terraform Resource ID.
- `settings` (Map of String) The key-value settings of the Frontend Observability app. Available Settings: `{combineLabData=(0|1),geolocation.level=(0|1),geolocation.level=0-4,geolocation.country_denylist=<comma-separated-list-of-country-codes>}`
- `stack_id` (Number) The Stack ID of the Grafana Cloud instance. Part of the Terraform Resource ID.

### Read-Only

- `collector_endpoint` (String) The collector URL Grafana Cloud Frontend Observability. Use this endpoint to send your Telemetry.
- `id` (Number) The Terraform Resource ID. This is auto-generated from Frontend Observability API.

## Import

Import is supported using the following syntax:

```shell
terraform import grafana_frontend_o11y_app.name "{{ stack_id }}:{{ name }}"
```
