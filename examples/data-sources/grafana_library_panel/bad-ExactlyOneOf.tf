resource "grafana_library_panel" "test_bad_inputs" {
  name = "test name"
  model_json = jsonencode({
    gridPos = {
      h = 8
      w = 12
    }
    id = 1
  })
}

data "grafana_library_panel" "bad_from_uid_id" {
  uid  = grafana_library_panel.test_bad_inputs.id
  name = grafana_library_panel.test_bad_inputs.name
}
