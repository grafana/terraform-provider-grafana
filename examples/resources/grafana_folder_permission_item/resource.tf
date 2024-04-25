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

resource "grafana_folder_permission_item" "on_role" {
  folder_uid = grafana_folder.collection.uid
  role       = "Viewer"
  permission = "Edit"
}

resource "grafana_folder_permission_item" "on_team" {
  folder_uid = grafana_folder.collection.uid
  team       = grafana_team.team.id
  permission = "View"
}

resource "grafana_folder_permission_item" "on_user" {
  folder_uid = grafana_folder.collection.uid
  user       = grafana_user.user.id
  permission = "Admin"
}

