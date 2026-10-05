terraform {
  required_providers {
    grafana = {
      source  = "grafana/grafana"
      version = "999.999.999"
    }
  }
}

provider "grafana" {
  url  = "http://localhost:3000"
  auth = "REDACTED"
}
