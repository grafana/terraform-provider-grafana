// create a minimal library panel inside the General folder
resource "grafana_library_panel" "test" {
  name       = "test name"
  folder_uid = "general"
  model_json = jsonencode({
    title   = "test name"
    type    = "text"
    version = 0
  })
}

data "grafana_library_panel" "from_name" {
  name = grafana_library_panel.test.name
}

data "grafana_library_panel" "from_uid" {
  uid = grafana_library_panel.test.uid
}

// create library panels to be added to a dashboard
resource "grafana_library_panel" "dashboard" {
  name       = "panel"
  folder_uid = "general"
  model_json = jsonencode({
    gridPos = {
      x = 0
      y = 0
      h = 10
    w = 10 }
    title = "panel"
    type  = "text"
  version = 0 })
}

// create a dashboard using the library panel
// `merge()` will add `libraryPanel` attribute to each library panel JSON
// Grafana will then connect any library panels found in dashboard JSON
resource "grafana_dashboard" "with_library_panel" {
  config_json = jsonencode({
    id = 12345
    panels = [
      merge(jsondecode(grafana_library_panel.dashboard.model_json), {
        libraryPanel = {
          name = grafana_library_panel.dashboard.name
          uid  = grafana_library_panel.dashboard.uid
        }
      })
    ]
    title         = "Production Overview"
    tags          = ["templated"]
    timezone      = "browser"
    schemaVersion = 16
    version       = 0
    refresh       = "25s"
  })
}

// dashboard_ids list attribute should contain dashboard id 12345
data "grafana_library_panel" "connected_to_dashboard" {
  uid = grafana_library_panel.dashboard.uid

  // the dashboard must be created before reading the library panel data
  depends_on = [grafana_dashboard.with_library_panel]
}
