resource "grafana_library_panel" "dashboard" {
  name       = "test name"
  model_json = jsonencode({
    gridPos  = {
      h      = 8,
      w      = 12 },
    id       = 1
  })
}

# make a dashboard wth a library panel
resource "grafana_dashboard" "test" {
  message          = "inital commit."
  config_json      = jsonencode({
    id             = 12345
    # panels         = [ merge(jsondecode(grafana_library_panel.dashboard.model_json), {
    #   libraryPanel = {
    #     uid        = grafana_library_panel.dashboard.id } }) ]
    title          = "Production Overview"
    tags           = [ "templated" ]
    timezone       = "browser"
    schemaVersion  = 16
    version        = 0
    refresh        = "25s"
  })
}
