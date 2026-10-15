resource "grafana_apps_folder_folder_v1" "example" {
  metadata {
    uid = "example-folder"
  }

  spec {
    title       = "Example Folder"
    description = "Managed by Terraform"
  }
}

# Nest a folder by pointing metadata.folder_uid at its parent.
resource "grafana_apps_folder_folder_v1" "nested" {
  metadata {
    uid        = "example-nested-folder"
    folder_uid = grafana_apps_folder_folder_v1.example.metadata.uid
  }

  spec {
    title = "Example Nested Folder"
  }
}

# A team folder: owned by a team, which changes how the folder is labelled and grouped
# in the UI. Team ownership does not grant any permissions on the folder -- use
# grafana_folder_permission_item for that.
resource "grafana_team" "example" {
  name = "Example Team"
}

resource "grafana_apps_folder_folder_v1" "team_owned" {
  metadata {
    uid = "example-team-folder"

    owner_references {
      api_version = "iam.grafana.app/v0alpha1"
      kind        = "Team"
      name        = grafana_team.example.team_uid
    }
  }

  spec {
    title = "Example Team Folder"
  }
}
