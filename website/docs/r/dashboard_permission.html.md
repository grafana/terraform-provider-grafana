---
layout: "grafana"
page_title: "Grafana: grafana_dashboard_permission"
sidebar_current: "docs-grafana-resource-dashboard-permission"
description: |-
  The grafana_dashboard_permission resource allows a Grafana dashboard's permissions to be maintained
---

# grafana\_dashboard\_permission

The dashboard permission resource allows permissions to be set for a given dashboard. Note: all permissions
must be specified for the given dashboard, including those for the default `Viewer` and `Editor` roles.

## Example Usage

```hcl
resource "grafana_team" "team" {
  name = "Team Name"
}

resource "grafana_user" "user" {
  email = "user.name@example.com"
}

resource "grafana_dashboard" "metrics" {
  config_json = file("grafana-dashboard.json")
}

resource "grafana_dashboard_permission" "collectionPermission" {
  dashboard_uid = grafana_dashboard.metrics.dashboard_id
  permissions {
    role       = "Editor"
    permission = "Edit"
  }
  permissions {
    team_id    = grafana_team.team.id
    permission = "View"
  }
  permissions {
    user_id    = grafana_user.user.id
    permission = "Admin"
  }
}
```

## Argument Reference

The following arguments are supported:

* `dashboard_id` - (Required) The ID of the dashboard
* `permissions` - (Required) The specified permission for the role, team, or user. 
                  `permissions` is described in more detail below. 

`permissions` supports the following:

The role, team, or user must be specified, but only one can be given for each 
`permissions` instance.

* `role` - (Optional) Used to control permissions for the `Editor` or `Viewer` roles
* `team_id` - (Optional) The ID of the team for which to control permissions
* `user_id` - (Optional) The ID of the user for which to control permissions
* `permission` - (Required) `View`, `Edit`, or `Admin` permissions

## Import

Dashboard permissions cannot be imported.
