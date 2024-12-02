resource "grafana_team" "team" {
  name = "Team Name"
}

resource "grafana_data_source" "test" {
  type                = "loki"
  name                = "loki-from-terraform"
  url                 = "https://mylokiurl.net"
  basic_auth_enabled  = true
  basic_auth_username = "username"

  json_data_encoded = jsonencode({
    authType          = "default"
    basicAuthPassword = "password"
  })
}

# resource "grafana_data_source_config_lbac_rules" "test_rule" {
#   datasource_uid = grafana_data_source.test.uid
#   rules = jsonencode({
#     "${grafana_team.team.team_uid}" = [
#       "{ cluster = \"dev-us-central-0\", namespace = \"hosted-grafana\" }",
#       "{ foo = \"qux\" }"
#     ]
#   })

#   depends_on = [
#     grafana_team.team,
#     grafana_data_source.test
#   ]
# }

