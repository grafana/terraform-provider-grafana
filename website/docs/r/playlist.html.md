---
layout: "grafana"
page_title: "Grafana: grafana_playlist"
sidebar_current: "docs-grafana-resource-playlist"
description: |-
  The grafana_playlist resource allows a Grafana playlist to be created.
---

# grafana\_playlist

The playlist resource allows a playlist to be created on a Grafana server.

## Example Usage

```hcl
resource "grafana_playlist" "playlist" {
  name     = "my playlist"
  interval = "5m"

  item {
    order = 1
    title = "Terraform Dashboard by Tag"
    type  = "dashboard_by_tag"
    value = "myTag"
  }

  item {
    order = 2
    title = "Terraform Dashbord By ID"
    type  = "dashboard_by_id"
    value = "myTag"
  }
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required, Forces new resource) The name of the playlist.
* `interval` - (Required) The amount of time for Grafana to stay on a particular dashboard before advancing to the next one on the playlist (e.g `5m`, `1h`)
* `item` - (Required) The Grafana dashboard(s) to add to the playlist.

`item` supports the following arguments:

* `order` - (Required) The order number in which the dashboard appears in the playlist (e.g. `1`, `2`)

* `title` - (Required) The name of the existing dashboard.

* `type` - (Optional) The description of the existing dashboard.

* `value` - (Optional) A tag associated with the dashboard.

## Attributes Reference

In addition to all arguments above, the following attributes are exported:

* `id` - The playlist ID.

## Import

Existing playlists can be imported using the `id` e.g.

```
$ terraform import grafana_playlist.playlist 123
```
