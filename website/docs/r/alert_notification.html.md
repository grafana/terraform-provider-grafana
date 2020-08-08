---
layout: "grafana"
page_title: "Grafana: grafana_alert_notification"
sidebar_current: "docs-grafana-alert-notification"
description: |-
  The grafana_alert_notification resource allows a Grafana Alert Notification channel to be created.
---

# grafana\_alert\_notification

The alert notification resource allows an alert notification channel to be created on a Grafana server.

## Example Usage

```hcl
resource "grafana_alert_notification" "email_someteam" {
  name = "Email that team"
  type = "email"
  is_default = false

  settings {
    "addresses" = "foo@example.net;bar@example.net"
    "uploadImage" = "false"
  }
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the alert notification channel.
* `type` - (Required) The type of the alert notification channel.
* `is_default` - (Optional) Is this the default channel for all your alerts.
* `settings` - (Optional) Additional settings, for full reference lookup [Grafana HTTP API documentation](http://docs.grafana.org/http_api/alerting).
* `disable_resolve_message` - (Optional) When selected, this option disables the resolve message [OK] that is sent when the alerting state returns to false (default: false).
* `send_reminder` - (Optional) If you want to send reminder alert. If this is in true, you must to specify a frequency. (default: false).
* `frequency` - (Optional) When this option is checked additional notifications (reminders) will be sent for triggered alerts. You can specify how often reminders should be sent using number of seconds (s), minutes (m) or hours (h), for example 30s, 3m, 5m or 1h (default: "").

**Note:** In `settings` the strings `"true"` and `"false"` are mapped to boolean `true` and `false` when sent to Grafana.

## Attributes Reference

The resource exports the following attributes:

* `id` - The ID of the resource
