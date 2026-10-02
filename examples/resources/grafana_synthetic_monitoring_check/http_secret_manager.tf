data "grafana_synthetic_monitoring_probes" "main" {}

resource "grafana_synthetic_monitoring_check" "http" {
  job     = "HTTP Secret Manager"
  target  = "https://api.example.com"
  enabled = true
  probes = [
    data.grafana_synthetic_monitoring_probes.main.probes.Mumbai,
  ]
  labels = {
    environment = "production"
    service     = "api"
  }
  settings {
    http {
      ip_version = "V4"
      method     = "GET"

      # All probes assigned to a check with secret_manager_enabled = true must
      # support protocol secrets, otherwise the API rejects the check.
      secret_manager_enabled = true

      # ${secrets.<name>} references are resolved from Grafana Secrets Manager at
      # check time. The leading $ is doubled so Terraform passes the reference
      # through literally instead of interpolating it.
      bearer_token = "$${secrets.my-api-token}"

      basic_auth {
        username = "admin"
        password = "$${secrets.my-api-password}"
      }

      headers = [
        "Accept: application/json",
        "User-Agent: Terraform-Synthetic-Monitoring",
      ]

      valid_status_codes = [
        200,
        201,
        202,
      ]

      fail_if_ssl     = false
      fail_if_not_ssl = true
    }
  }
}
