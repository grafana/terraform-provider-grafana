resource "grafana_apps_secret_keeper_activation_v1beta1" "active" {
  metadata {
    uid = grafana_apps_secret_keeper_v1beta1.aws_production.metadata.uid
  }
}
