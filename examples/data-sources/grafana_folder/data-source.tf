resource "grafana_folder" "test" {
  title = "test-folder"
}

data "grafana_folder" "from_title" {
  title = grafana_folder.test.title
}

data "grafana_folder" "from_uid" {
  uid = grafana_folder.test.uid
}

data "grafana_folder" "from_id" {
  id = grafana_folder.test.id
}
