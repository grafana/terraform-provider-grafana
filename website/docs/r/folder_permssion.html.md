---
layout: "grafana"
page_title: "Grafana: grafana_folder_permission"
sidebar_current: "docs-grafana-resource-folder-permission"
description: |-
  The grafana_folder_permission resource allows a Grafana folder's permisions to be maintained
---

# grafana\_folder\_permission

The folder permission resource allows permissions to be set for a given folder.

## Example Usage

```hcl
resource "grafana_team" "team" {
  name = "Team Name"
}

resource "grafana_user" "user" {
  email = "user.name@example.com"
}

resource "grafana_folder" "collection" {
  title = "Folder Title"
}

resource "grafana_dashboard_permission" "collectionPermission" {
  folder_uid = "${grafana_folder.colection.uid}"
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

* `folder_uid` - (Required) The UID of the folder
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

Folder permissions cannot be imported.
