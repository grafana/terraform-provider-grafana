resource "grafana_library_panel" "test" {
  name = "panel"
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
