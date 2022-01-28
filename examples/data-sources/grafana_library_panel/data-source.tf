resource "grafana_library_panel" "test" {
  name = "test name"
  model_json = jsonencode({})
}

data "grafana_library_panel" "from_name" {
  name = "test name"
}

data "grafana_library_panel" "from_uid" {
  uid = grafana_library_panel.test.id
}
