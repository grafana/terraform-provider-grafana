resource "grafana_folder" "data_source_dashboards" {
  title = "test folder data_source_dashboards"
}

// retrieve dashboards by tags, folderIDs, or both
resource "grafana_dashboard" "data_source_dashboards1" {
  folder = grafana_folder.data_source_dashboards.id
  config_json = jsonencode({
    id            = 23456
    title         = "data_source_dashboards 1"
    tags          = ["data_source_dashboards"]
    timezone      = "browser"
    schemaVersion = 16
  })
}

data "grafana_dashboards" "tags" {
  tags = jsondecode(grafana_dashboard.data_source_dashboards1.config_json)["tags"]
}

data "grafana_dashboards" "folder_ids" {
  folder_ids = [grafana_dashboard.data_source_dashboards1.folder]
}

data "grafana_dashboards" "folder_ids_tags" {
  folder_ids = [grafana_dashboard.data_source_dashboards1.folder]
  tags       = jsondecode(grafana_dashboard.data_source_dashboards1.config_json)["tags"]
}

resource "grafana_dashboard" "data_source_dashboards2" {
  folder = 0 // General folder
  config_json = jsonencode({
    id            = 23456
    title         = "data_source_dashboards 2"
    tags          = ["prod"]
    timezone      = "browser"
    schemaVersion = 16
  })
}

// use depends_on to wait for dashboard resource to be created before searching
data "grafana_dashboards" "all" {
  depends_on = [
    grafana_dashboard.data_source_dashboards1,
    grafana_dashboard.data_source_dashboards2
  ]
}

data "grafana_dashboard" "from_data_source" {
  uid = data.grafana_dashboards.all.dashboards[0].uid
}
