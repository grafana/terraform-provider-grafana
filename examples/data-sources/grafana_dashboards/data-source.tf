resource "grafana_folder" "test" {
  title = "test folder 1"
}

resource "grafana_dashboard" "test1" {
  folder = 0
  config_json = jsonencode({
    title         = "Production Overview 1",
    tags          = ["templated"],
    timezone      = "browser",
    schemaVersion = 16,
  })
}

resource "grafana_dashboard" "test2" {
  folder = grafana_folder.test.id
  config_json = jsonencode({
    title         = "Production Overview 2",
    tags          = ["templated"],
    timezone      = "browser",
    schemaVersion = 16,
  })
}

data "grafana_dashboards" "with_folder_id" {
  folder_ids = [grafana_folder.test.id]
}

data "grafana_dashboards" "all" {
}
