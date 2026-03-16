resource "grafana_apps_secret_keeper_v1beta1" "aws_production" {
  metadata {
    uid = "aws-production"
  }
  spec {
    description = "AWS Production"

    aws {
      region = "us-east-1"

      assume_role {
        assume_role_arn = "arn:aws:iam::123456789012:role/GrafanaSecretsAccess"
        external_id     = "grafana-unique-external-id"
      }
    }
  }
}
