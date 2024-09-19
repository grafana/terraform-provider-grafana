terraform {
  required_providers {
    grafana = {
      source  = "grafana/grafana"
      version = "999.999.999"
    }
  }
}

provider "grafana" {
  url             = "https://tfprovidertests.grafana.net/"
  auth            = "REDACTED"
  sm_url          = "https://synthetic-monitoring-api-us-east-0.grafana.net"
  sm_access_token = "REDACTED"
}
