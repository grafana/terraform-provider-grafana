resource "grafana_folder" "data_source_folders" {
  title = "data_source_folders1"
}

resource "grafana_folder" "data_source_folders" {
  title = "data_source_folders2"
}

data "grafana_folders" "all" {
}

data "grafana_folders" "one" {
  limit = 1
}
