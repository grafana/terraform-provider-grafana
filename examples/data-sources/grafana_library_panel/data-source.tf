resource "grafana_library_panel" "test" {
  name       = "test name"
  model_json = jsonencode({
    gridPos  = {
      h      = 8
      w      = 12 }
    id       = 1
    # if not set, Grafana v8.0/v8.1 will error "inconsistent final plan" in dashboard resource
    title    = "test name"
  })
}

data "grafana_library_panel" "from_name" {
  name = grafana_library_panel.test.name
}

data "grafana_library_panel" "from_uid" {
  uid = grafana_library_panel.test.id
}
