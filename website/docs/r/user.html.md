---
layout: "grafana"
page_title: "Grafana: grafana_user"
sidebar_current: "docs-grafana-user"
description: |-
  The grafana_user resource allows a Grafana user to be created.
---

# grafana\_user

The user resource allows Grafana user management.

~> **NOTE on `grafana_user`:** - this resource uses Grafana's admin APIs for
creating and updating users. This API does not currently work with API Tokens.
You must use Basic Authentication (username and password).

## Example Usage

```hcl
resource "grafana_user" "staff" {
  email    = "staff.name@example.com"
  name     = "Staff Name"
  login    = "staff"
  password = "my-password"
}
```

## Argument Reference

The following arguments are supported:

* `email` - (Required) The email address of the Grafana user.
* `name` - (Optional) The display name for the Grafana user.
* `login` - (Optional) The username for the Grafana user.
* `password` - (Optional) The password for the Grafana user.

## Attributes Reference

The following attributes are exported:

* `id` - The ID of the resource
