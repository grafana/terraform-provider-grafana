terraform {
  required_providers {
    grafana = {
      version = "999.999.999"
      source  = "grafana/grafana"
    }
  }
}

provider "grafana" {
  url  = "http://localhost:3000"
  auth = "admin:admin"
}

resource "grafana_team" "my_team" {
  name = "My Team"
}

resource "grafana_folder" "my_folder" {
  title = "My Folder"
}

// This resource is implemented by the Terraform Plugin Framework
// We're testing that both the legacy SDK (folder resource) and the new SDK/Plugin Framework work correctly
resource "grafana_folder_permission_item" "my_folder_permission" {
  folder_uid = grafana_folder.my_folder.uid
  team       = grafana_team.my_team.id
  permission = "Edit"
}
