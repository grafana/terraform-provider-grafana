resource "grafana_folder" "test" {
  title = "test folder 1"
}

resource "grafana_dashboard" "test1" {
  folder = 0  // General folder
  config_json = jsonencode({
    id            = 12345
    title         = "Production Overview 1"
    tags          = ["dev"]
    timezone      = "browser"
    schemaVersion = 16
  })
}

resource "grafana_dashboard" "test2" {
  folder = grafana_folder.test.id
  config_json = jsonencode({
    id            = 23456
    title         = "Production Overview 2"
    tags          = ["prod"]
    timezone      = "browser"
    schemaVersion = 16
  })
}

/* data "grafana_dashboards" "with_folder_id" {
  folder_ids = [grafana_folder.test.id]
} */

data "grafana_dashboards" "with_tags" {
  tags = ["prod"]
}

data "grafana_dashboards" "all" {
}
