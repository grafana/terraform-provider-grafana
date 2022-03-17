resource "grafana_folder" "test" {
  title = "test-folder"
  uid   = "test-ds-folder-uid"
}

data "grafana_folder" "from_title" {
  title = grafana_folder.test.title
}
