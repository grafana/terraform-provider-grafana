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

resource "grafana_cloud_access_policy_rotating_token" "test" {
  region                = "prod-us-east-0"
  access_policy_id      = grafana_cloud_access_policy.test.policy_id
  name_prefix           = "my-policy-rotating-token"
  display_name          = "My Policy Rotating Token"
  expire_after          = "720h"
  early_rotation_window = "24h"
}
