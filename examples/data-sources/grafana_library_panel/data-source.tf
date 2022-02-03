resource "grafana_library_panel" "test" {
  name        = "test name"
  folder_id   = 0
  model_json  = jsonencode({
    title     = "test name"
    type      = "text"
    version   = 0
  })
}

data "grafana_library_panel" "from_name" {
  name = grafana_library_panel.test.name
}

data "grafana_library_panel" "from_uid" {
  uid = grafana_library_panel.test.id
}
