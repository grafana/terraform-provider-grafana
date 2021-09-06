resource "grafana_folder" "collection" {
  title = "Folder Title"
}

resource "grafana_dashboard" "metrics" {
  folder      = grafana_folder.collection.id
  config_json = file("grafana-dashboard.json")
}
