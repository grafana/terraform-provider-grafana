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

# data "grafana_library_panel" "from_name" {
#   name = "test name"
# }

# data "grafana_library_panel" "from_uid" {
#   uid = grafana_library_panel.test.id
# }


# make a dashboard wth a library panel

resource "grafana_dashboard" "test" {
  message          = "inital commit."
  config_json      = jsonencode({
    id             = 12345,
    panels         = [ jsondecode(grafana_library_panel.test.model_json) ]
    # panels         = [ merge(jsondecode(grafana_library_panel.test.model_json), {
      # libraryPanel = {
      #   uid        = grafana_library_panel.test.id } }) ]
    title          = "Production Overview",
    tags           = [ "templated" ],
    timezone       = "browser",
    schemaVersion  = 16,
    version        = 0,
    refresh        = "25s"
  })
}
