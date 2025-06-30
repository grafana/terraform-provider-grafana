# grafana_scim_config

Manages Grafana SCIM configuration.

**Note:** This resource is available only with Grafana Enterprise.

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

## Notes

* The SCIM configuration is stored in the namespace `stacks-{stackID}` using the stack ID from the provider configuration.
* This resource requires Grafana Enterprise with SCIM features enabled. 