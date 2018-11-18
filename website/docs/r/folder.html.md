---
layout: "grafana"
page_title: "Grafana: grafana_folder"
sidebar_current: "docs-grafana-resource-folder"
description: |-
  The grafana_folder resource allows a Grafana folder to be created.
---

# grafana\_folder

The folder resource allows a folder to be created on a Grafana server.

## Example Usage

```hcl
resource "grafana_folder" "examples" {
  title = "Example dashboards"
}
```

Folders are a way to organize and group dashboards - very useful if you
have a lot of dashboards or multiple teams using the same Grafana instance.

## Argument Reference

The following arguments are supported:

* `title` - (Required) The name for the folder.

## Attributes Reference

The resource exports the following attributes:

* `id` - The opaque unique id assigned to the folder by the Grafana
  server.
* uid` - An external id of the folder in Grafana (stable when folders are
  migrated between Grafana instances). The `uid` is required by several Grafana
  Folder APIs.
