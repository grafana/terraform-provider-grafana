---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "grafana_synthetic_monitoring_check_alerts Resource - terraform-provider-grafana"
subcategory: "Synthetic Monitoring"
description: |-
  Manages alerts for a check in Grafana Synthetic Monitoring.
  Official documentation https://grafana.com/docs/grafana-cloud/testing/synthetic-monitoring/configure-alerts/configure-per-check-alerts/
---

# grafana_synthetic_monitoring_check_alerts (Resource)

Manages alerts for a check in Grafana Synthetic Monitoring.

* [Official documentation](https://grafana.com/docs/grafana-cloud/testing/synthetic-monitoring/configure-alerts/configure-per-check-alerts/)

## Example Usage

```terraform
resource "grafana_synthetic_monitoring_check" "main" {
  job     = "Check Alert Test"
  target  = "https://grafana.com"
  enabled = true
  probes  = [1]
  labels  = {}
  settings {
    http {
      ip_version = "V4"
      method     = "GET"
    }
  }
}

resource "grafana_synthetic_monitoring_check_alerts" "main" {
  check_id = grafana_synthetic_monitoring_check.main.id
  alerts = [{
    name      = "ProbeFailedExecutionsTooHigh"
    threshold = 1
    period    = "15m"
    },
    {
      name      = "TLSTargetCertificateCloseToExpiring"
      threshold = 14
      period    = ""
  }]
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `alerts` (Set of Object) List of alerts for the check. (see [below for nested schema](#nestedatt--alerts))
- `check_id` (Number) The ID of the check to manage alerts for.

### Read-Only

- `id` (String) The ID of this resource.

<a id="nestedatt--alerts"></a>
### Nested Schema for `alerts`

Required:

- `name` (String)
- `period` (String)
- `threshold` (Number)

## Import

Import is supported using the following syntax:

```shell
terraform import grafana_synthetic_monitoring_check_alerts.name "{{ check_id }}"
```
