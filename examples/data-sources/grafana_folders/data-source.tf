resource "grafana_folder" "data_source_folders1" {
  title = "data_source_folders1"
}

resource "grafana_folder" "data_source_folders2" {
  title = "data_source_folders2"
}

// wait for folder resources to be created before searching
data "grafana_folders" "all" {
  depends_on = [
    grafana_folder.data_source_folders1,
    grafana_folder.data_source_folders2,
  ]
}

data "grafana_folders" "one" {
  limit = 1
  depends_on = [
    grafana_folder.data_source_folders1,
    grafana_folder.data_source_folders2,
  ]
}

// test to make sure search worked
data "grafana_folder" "test" {
  uid = data.grafana_folders.all.folders["data_source_folders1"]
}
