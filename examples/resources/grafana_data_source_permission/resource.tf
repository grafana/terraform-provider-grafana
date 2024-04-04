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

resource "grafana_data_source_permission" "fooPermissions" {
  datasource_uid = grafana_data_source.foo.uid
  permissions {
    team_id    = grafana_team.team.id
    permission = "Edit"
  }
  permissions {
    user_id    = grafana_user.user.id
    permission = "Edit"
  }
  permissions {
    built_in_role = "Viewer"
    permission    = "Query"
  }
  permissions {
    user_id    = grafana_service_account.sa.id
    permission = "Query"
  }
}
