resource "grafana_folder" "test_folder" {
  title = "Terraform Folder Folder ID Test"
  uid   = "folder-dashboard-id-test"
}

resource "grafana_dashboard" "test_folder" {
  folder = grafana_folder.test_folder.id
  config_json = jsonencode({
    "title" : "Terraform Folder Test Dashboard With ID",
    "id" : 123,
    "version" : "434",
    "uid" : "folder-dashboard-test-ref-with-id"
  })
}

