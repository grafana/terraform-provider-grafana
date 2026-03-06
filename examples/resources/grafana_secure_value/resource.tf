resource "grafana_apps_secret_securevalue_v1beta1" "db_password" {
  metadata {
    uid = "db-password"
  }
  spec {
    description = "Database password"
    value       = "change-me"
    decrypters  = ["grafana", "k6"]
  }
}

resource "grafana_apps_secret_securevalue_v1beta1" "external_api_key" {
  metadata {
    uid = "external-api-key"
  }
  spec {
    description = "External API key"
    ref         = "path/to/existing/secret"
    decrypters  = ["grafana"]
  }
}
