resource "grafana_library_panel" "test" {
  name = "panelname"
  model_json = jsonencode({
    title       = "test name"
    type        = "text"
    version     = 0
    description = "test description"
  })
}

resource "grafana_folder" "test" {
  title = "Panel Folder"
  uid   = "panelname-folder"
}

resource "grafana_library_panel" "folder" {
  name       = "panelname In Folder"
  folder_uid = grafana_folder.test.uid
  model_json = jsonencode({
    gridPos = {
      x = 0
      y = 0
      h = 10
      w = 10
    }
    title   = "panel"
    type    = "text"
    version = 0
  })
}

data "grafana_library_panels" "all" {
  depends_on = [grafana_library_panel.folder, grafana_library_panel.test]
}
