resource "grafana_dashboard" "metrics" {
  config_json = file("grafana-dashboard.json")
}
