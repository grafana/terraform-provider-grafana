resource "grafana_folder" "test_a" {
  title = "test-folder-a"
  uid   = "test-ds-folder-uid-a"
}

resource "grafana_folder" "test_b" {
  title = "test-folder-b"
  uid   = "test-ds-folder-uid-b"
}

data "grafana_folders" "test" {
  depends_on = [
    grafana_folder.test_a,
    grafana_folder.test_b,
  ]
}
