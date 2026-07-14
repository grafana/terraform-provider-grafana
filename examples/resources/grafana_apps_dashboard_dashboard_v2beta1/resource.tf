resource "grafana_apps_dashboard_dashboard_v2beta1" "example" {
  metadata {
    uid = "example-dashboard-v2"
  }

  spec {
    title = "Example Dashboard V2"
    json = jsonencode({
      title       = "Example Dashboard V2"
      cursorSync  = "Off"
      elements    = {}
      layout      = { kind = "GridLayout", spec = { items = [] } }
      links       = []
      preload     = false
      annotations = []
      variables   = []
      timeSettings = {
        timezone = "browser"
        from     = "now-6h"
        to       = "now"
      }
    })
  }
}
