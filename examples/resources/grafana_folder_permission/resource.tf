resource "grafana_team" "team" {
  name = "Team Name"
}

resource "grafana_user" "user" {
  email    = "user.name@example.com"
  login    = "user.name"
  password = "my-password"
}

resource "grafana_folder" "collection" {
  title = "Folder Title"
}

resource "grafana_folder_permission" "collectionPermission" {
  folder_uid = grafana_folder.collection.uid
  permissions {
    role       = "Editor"
    permission = "Edit"
  }
  permissions {
    team_id    = grafana_team.team.id
    permission = "View"
  }
  permissions {
    user_id    = grafana_user.user.id
    permission = "Admin"
  }
}
