resource "grafana_team" "team" {
  name = "Team Name"
}

resource "grafana_data_source" "foo" {
  type = "cloudwatch"
  name = "cw-example"

  json_data_encoded = jsonencode({
    defaultRegion = "us-east-1"
    authType      = "keys"
  })

  secure_json_data_encoded = jsonencode({
    accessKey = "123"
    secretKey = "456"
  })
}

resource "grafana_user" "user" {
  name     = "test-ds-permissions"
  email    = "test-ds-permissions@example.com"
  login    = "test-ds-permissions"
  password = "hunter2"
}

resource "grafana_service_account" "sa" {
  name = "test-ds-permissions"
  role = "Viewer"
}

resource "grafana_data_source_permission_item" "team" {
  datasource_uid = grafana_data_source.foo.uid
  team           = grafana_team.team.id
  permission     = "Edit"
}

resource "grafana_data_source_permission_item" "user" {
  datasource_uid = grafana_data_source.foo.uid
  user           = grafana_user.user.id
  permission     = "Edit"
}

resource "grafana_data_source_permission_item" "role" {
  datasource_uid = grafana_data_source.foo.uid
  role           = "Viewer"
  permission     = "Query"
}

resource "grafana_data_source_permission_item" "service_account" {
  datasource_uid = grafana_data_source.foo.uid
  user           = grafana_service_account.sa.id
  permission     = "Query"
}

