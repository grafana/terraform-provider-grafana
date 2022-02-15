resource "grafana_folder" "data_source_folders1" {
  title = "data_source_folders1"
}

resource "grafana_folder" "data_source_folders2" {
  title = "data_source_folders2"
}

data "grafana_folders" "all" {
}

data "grafana_folders" "one" {
  limit = 1
}

// test to make sure it worked
data "grafana_folder" "test" {
  uid = data.grafana_folders.all["data_source_folders1"].uid
}
