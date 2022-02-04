resource "grafana_folder" "test" {
  title = "test-folder"
}

data "grafana_folder" "from_title" {
  title = grafana_folder.test.title
}
