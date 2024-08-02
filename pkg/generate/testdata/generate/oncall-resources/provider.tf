terraform {
  required_providers {
    grafana = {
      source  = "grafana/grafana"
      version = "999.999.999"
    }
  }
}

provider "grafana" {
  url                 = "https://tfprovidertests.grafana.net/"
  auth                = "REDACTED"
  oncall_url          = "https://oncall-prod-us-central-0.grafana.net/oncall"
  oncall_access_token = "REDACTED"
}
