---
layout: "grafana"
page_title: "Grafana: grafana_team"
sidebar_current: "docs-grafana-resource-team"
description: |-
  The grafana_team resource allows a Grafana team to be created.
---

# grafana\_team

The team resource allows Grafana teams and their membership to
be created and managed.

## Example Usage

```hcl
# Create a Grafana team with defined membership. The resource
# requires users to already exist in the system
resource "grafana_team" "test-team" {
  name    = "Test Team"
  email   = "teamemail@example.com"
  members = [
    "viewer-01@example.com"
  ]
}
```


## Argument Reference

The following arguments are supported:

* `name` - (Required) The display name for the Grafana team created.

* `email` - (Optional) An email address for the team.

* `members` - (Optional) A list of email addresses corresponding to users who
  should be given membership to the team. Note: users specified here must already 
  exist in Grafana.

Note - Users should be listed in ascending alphabetical order (A-Z). By defining 
users in alphabetical order, Terraform is prevented from detecting unnecessary changes
when comparing the list of defined users in the resource to the (ordered) list
returned by the Grafana API.

## Attributes Reference

The following attributes are exported:

* `team_id` - The team id assigned to this team by Grafana.

## Import

Existing teams can be imported using the team id. Currently this value is only
available via the [`http api`](https://grafana.com/docs/grafana/latest/http_api/team/). 

```
$ terraform import grafana_team.name {team_id}
```
