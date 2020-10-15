---
layout: "grafana"
page_title: "Grafana: grafana_organization"
sidebar_current: "docs-grafana-resource-organization"
description: |-
  The grafana_organization resource allows a Grafana organization to be created.
---

# grafana\_organization

The organization resource allows Grafana organizations and their membership to
be created and managed.

## Example Usage

```hcl
# Create a Grafana organization with defined membership, creating placeholder
# accounts for users that don't exist.
resource "grafana_organization" "test-org" {
  name         = "Test Organization"
  admin_user   = "admin"
  create_users = true
  admins = [
    "admin@example.com"
  ]
  editors = [
    "editor-01@example.com",
    "editor-02@example.com"
  ]
  viewers = [
    "viewer-01@example.com",
    "viewer-02@example.com"
  ]
}
```


## Argument Reference

The following arguments are supported:

* `name` - (Required) The display name for the Grafana organization created.

* `admin_user` - (Optional) The login name of the configured
  [default admin user](http://docs.grafana.org/installation/configuration/#admin-user)
  for the Grafana installation. If unset, this value defaults to `admin`, the
  Grafana default. Grafana adds the default admin user to all organizations
  automatically upon creation, and this parameter keeps Terraform from removing
  it from organizations.

* `create_users` - (Optional) Whether or not to create Grafana users specified
  in the organization's membership if they don't already exist in Grafana. If
  unspecified, this parameter defaults to `true`, creating placeholder users
  with the `name`, `login`, and `email` set to the email of the user, and a
  random password. Setting this option to `false` will cause an error to be
  thrown for any users that do not already exist in Grafana.

  This option is particularly useful when integrating Grafana with external
  authentication services such as
  [`auth.github`](http://docs.grafana.org/installation/configuration/#auth-github)
  and
  [`auth.google`](http://docs.grafana.org/installation/configuration/#auth-google).

* `admins` - (Optional) A list of email addresses corresponding to users who
  should be given `admin` access to the organization. Note: users specified
  here must already exist in Grafana unless 'create_users' is set to true.

* `editors` - (Optional) A list of email addresses corresponding to users who
  should be given `editor` access to the organization. Note: users specified
  here must already exist in Grafana unless 'create_users' is set to true.

* `viewers` - (Optional) A list of email addresses corresponding to users who
  should be given `viewer` access to the organization. Note: users specified
  here must already exist in Grafana unless 'create_users' is set to true.

A user can only be listed under one role-group for an organization, listing the
same user under multiple roles will cause an error to be thrown.

Note - Users specified for each role-group (`admins`, `editors`, `viewers`)
should be listed in ascending alphabetical order (A-Z). By defining users in
alphabetical order, Terraform is prevented from detecting unnecessary changes
when comparing the list of defined users in the resource to the (ordered) list
returned by the Grafana API.

## Attributes Reference

The following attributes are exported:

* `org_id` - The organization id assigned to this organization by Grafana.

## Import

Existing organizations can be imported using the organization id obtained from
the Grafana Web UI under 'Server Admin'.

```
$ terraform import grafana_organization.org_name {org_id}
```
