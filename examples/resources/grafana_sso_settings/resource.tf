resource "grafana_sso_settings" "github_sso_settings" {
  provider_name = "github"
  settings {
    client_id = "github_client_id"
    client_secret = "github_client_secret"
    team_ids = "12,50,123"
    allowed_organizations = "organization1,organization2"
  }
}
