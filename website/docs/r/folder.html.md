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
resource "grafana_folder" "collection" {
  title = "Folder Title"
}

resource "grafana_dashboard" "dashboard_in_folder" {
  folder = "${grafana_folder.collection.id}"
  ...
}
```

## Argument Reference

The following arguments are supported:

* `title` - (Required) The title of the folder.

## Attributes Reference

The resource exports the following attributes:

* `id` - The internal id of the folder in Grafana (only guaranteed to be unique
  within this Grafana instance). The `id` is used by the `grafana_dashboard` resource
  to place a dashboard within a folder.
* `uid` - An external id of the folder in Grafana (stable when folders are migrated
  between Grafana instances). The `uid` is required by several Grafana Folder APIs.

## Import

Folders cannot be imported.
