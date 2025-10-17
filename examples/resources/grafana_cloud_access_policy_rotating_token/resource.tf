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

resource "time_rotating" "token_rotation" {
  rotation_days = 30
}

resource "grafana_cloud_access_policy_rotating_token" "test" {
  region                 = "prod-us-east-0"
  access_policy_id       = grafana_cloud_access_policy.test.policy_id
  name_prefix            = "my-policy-rotating-token"
  display_name           = "My Policy Rotating Token"
  rotate_after           = time_rotating.token_rotation.unix
  post_rotation_lifetime = "24h"
}
