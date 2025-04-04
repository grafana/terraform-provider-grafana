---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "grafana_frontend_o11y_app Data Source - terraform-provider-grafana"
subcategory: "Frontend Observability"
description: |-
  
---

# grafana_frontend_o11y_app (Data Source)



## Example Usage

```terraform
data "grafana_cloud_stack" "teststack" {
  provider = grafana.cloud
  name     = "gcloudstacktest"
}

data "grafana_frontend_o11y_app" "test-app" {
  provider = grafana.cloud
  stack_id = data.grafana_cloud_stack.teststack.id
  name     = "test-app"
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `name` (String) The name of the Frontend Observability App. Part of the Terraform Resource ID.
- `stack_id` (Number) The Stack ID of the Grafana Cloud instance. Part of the Terraform Resource ID.

### Read-Only

- `allowed_origins` (List of String) A list of allowed origins for CORS.
- `collector_endpoint` (String) The collector URL Grafana Cloud Frontend Observability. Use this endpoint to send your Telemetry.
- `extra_log_attributes` (Map of String) The extra attributes to append in each signal.
- `id` (Number) The Terraform Resource ID. This auto-generated from Frontend Observability API.
- `settings` (Map of String) The settings of the Frontend Observability App.
