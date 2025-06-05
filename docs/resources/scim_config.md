# grafana_scim_config

Manages Grafana SCIM configuration using the new app platform APIs.

## Example Usage

```hcl
resource "grafana_scim_config" "default" {
  enable_user_sync  = true
  enable_group_sync = false
}
```

## Argument Reference

* `enable_user_sync` (Required) - Whether user synchronization is enabled.
* `enable_group_sync` (Required) - Whether group synchronization is enabled.

## Attribute Reference

* `id` - The ID of the SCIM config resource. This should always be `default`.

## Import

SCIM config can be imported using the resource name, e.g.,

```
terraform import grafana_scim_config.default scim-config
``` 