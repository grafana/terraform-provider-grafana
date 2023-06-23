resource "grafana_folder" "test_folder" {
  title = "Terraform Folder Folder UID Test"
  uid   = "folder-dashboard-uid-test"
}

resource "grafana_dashboard" "test_folder" {
  folder = grafana_folder.test_folder.uid
  config_json = jsonencode({
    "title" : "Terraform Folder Test Dashboard With UID",
    "id" : 1234,
    "version" : "4345",
    "uid" : "folder-dashboard-test-ref-with-uid"
  })
}
