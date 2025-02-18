data "grafana_cloud_organization" "current" {
  slug = "<your org slug>"
}

resource "grafana_cloud_access_policy" "test" {
  region       = "prod-us-east-0"
  name         = "my-policy"
  display_name = "My Policy"

  scopes = ["metrics:read", "logs:read"]

  realm {
    type       = "org"
    identifier = data.grafana_cloud_organization.current.id

    label_policy {
      selector = "{namespace=\"default\"}"
    }
  }
}

resource "grafana_cloud_access_policy_token" "test" {
  region           = "prod-us-east-0"
  access_policy_id = grafana_cloud_access_policy.test.policy_id
  name             = "my-policy-token"
  display_name     = "My Policy Token"
  expires_at       = "2023-01-01T00:00:00Z"
}
