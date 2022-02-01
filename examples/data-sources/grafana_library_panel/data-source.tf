provider "grafana" {
  url  = "http://enverus.grafana.net/"
  auth = "eyJrIjoiNmFyaUR1YzFZbkRkMFdPZkNKM0hyN01odFAyaDdNcU8iLCJuIjoib25wcmVtLWltcG9ydC1kYXNoYm9hcmRzIiwiaWQiOjF9"
}

resource "grafana_library_panel" "test" {
  for_each    = {
    1 = { gridPos = {h = 8, w = 12}, id = 1, title = "test name 1", type = "text" },
    2 = { gridPos = {h = 8, w = 12}, id = 2, title = "test name 2", type = "text" }, }

  name        = each.value.title
  model_json  = jsonencode(each.value)
}

# data "grafana_library_panel" "from_name" {
#   name = "test name"
# }

# data "grafana_library_panel" "from_uid" {
#   uid = grafana_library_panel.test.id
# }

# add libraryPanel object to each panel before creating dashboard.
# Grafana will link panels to dashboards using uid and name when creating the dashboard.
locals {
  panels = [ for this_p in grafana_library_panel.test : merge(this_p, {
    libraryPanel = {
      uid        = this_p.id
      name       = this_p.name } }) ]
}

# make a dashboard wth a library panel
resource "grafana_dashboard" "test" {
  message          = "inital commit."
  config_json      = jsonencode({
    id             = 12345
    panels         = local.panels
    title          = "Production Overview"
    tags           = [ "templated" ]
    timezone       = "browser"
    schemaVersion  = 16
    version        = 0
    refresh        = "25s"
  })
}
