resource "grafana_folder" "test_folder" {
  title = "Terraform Folder Test Folder"
}

resource "grafana_library_panel" "test_folder" {
  name          = "test-folder"
  folder_id     = grafana_folder.test_folder.id
  model_json    = jsonencode({
    title       = "test-folder",
    id          = 12,
    type        = "dash-db",
    version     = 43,
  })
}
