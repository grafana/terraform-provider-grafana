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

      # The ${secrets.<name>} reference is what turns secret manager on, so there
      # is no flag to set. It is resolved from Grafana Secrets Manager at check
      # time, and the leading $ is doubled so Terraform passes it through
      # literally. All probes assigned to the check must support protocol secrets.
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
