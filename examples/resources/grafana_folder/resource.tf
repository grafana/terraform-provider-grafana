resource "grafana_folder" "test_folder" {
  title = "Terraform Test Folder"
}

resource "grafana_dashboard" "test_folder" {
  folder      = grafana_folder.test_folder.id
  config_json = <<EOD
{
  "title": "Dashboard in folder",
  "uid": "dashboard-in-folder"
}
EOD
}

resource "grafana_folder" "test_folder_with_uid" {
  uid   = "test-folder-uid"
  title = "Terraform Test Folder With UID"
}
