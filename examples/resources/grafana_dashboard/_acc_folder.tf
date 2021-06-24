resource "grafana_folder" "test_folder" {
  title = "Terraform Folder Test Folder"
}

resource "grafana_dashboard" "test_folder" {
  folder      = grafana_folder.test_folder.id
  config_json = <<EOD
{
  "title": "Terraform Folder Test Dashboard",
  "id": 12,
  "version": "43",
  "uid": "folder"
}
EOD
}
