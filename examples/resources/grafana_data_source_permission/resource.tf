resource "grafana_team" "team" {
  name = "Team Name"
}

resource "grafana_data_source" "foo" {
  type = "cloudwatch"
  name = "cw-example"

  json_data {
    default_region = "us-east-1"
    auth_type      = "keys"
  }

  secure_json_data {
    access_key = "123"
    secret_key = "456"
  }
}

resource "grafana_data_source_permission" "fooPermissions" {
  datasource_id = grafana_data_source.foo.id
  permissions {
    team_id    = grafana_team.team.id
    permission = "Query"
  }
  permissions {
    user_id    = 3 // 3 is the admin user in cloud. It can't be queried
    permission = "Query"
  }
  // As the data_source_permission is using legacy implementation, this temporary fixes the issue when Grafana returns more on-the-fly computed permissions than provisioned.
  permissions {
    permission = "Query"
  }
}
