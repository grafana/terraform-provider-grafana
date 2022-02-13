// create a minimal library panel inside the General folder
resource "grafana_library_panel" "datasource_library_panels" {
  name       = "datasource_library_panels"
  folder_id  = 0
  model_json = jsonencode({
    title    = "datasource_library_panels"
    type     = "text"
    version  = 0
  })
}

// search for all library panels after waiting for the panel above to be created
data "grafana_library_panels" "datasource_library_panels" {
  depends_on = [grafana_library_panel.datasource_library_panels]
}

// get panel information by its UID
data "grafana_library_panel" "datasource_library_panels" {
  uid = data.grafana_library_panels.datasource_library_panels.panels[0].uid
}

// make a new folder to copy the library panel into
resource "grafana_folder" "datasource_library_panels_moved" {
  title = "datasource_library_panels"
}

// duplicate this library panel into the new folder by copying the panel's JSON model
resource "grafana_library_panel" "datasource_library_panels_moved" {
  name        = "datasource_library_panels_moved"
  folder_id   = grafana_folder.datasource_library_panels_moved.id
  model_json  = data.grafana_library_panel.datasource_library_panels.model_json
}

// search for this new panel by the (newly created) folder id
data "grafana_library_panels" "datasource_library_panels_moved" {
  folder_ids = [grafana_library_panel.datasource_library_panels_moved.folder_id]
}

// get all panel information by its UID (to test it worked successfully)
data "grafana_library_panel" "datasource_library_panels_moved" {
  uid = data.grafana_library_panels.datasource_library_panels_moved.panels[0].uid
}
