resource "grafana_library_panel" "metrics" {
  config_json = file("grafana-library-panel.json")
}
