terraform {
  required_providers {
    grafana = {
      source = "grafana/grafana"
    }
  }
}

provider "grafana" {
  url  = var.grafana_url
  auth = var.grafana_auth

  # Grafana API server TLS configuration
  insecure_skip_verify = var.grafana_tls_ca == "" ? true : false
  ca_cert              = var.grafana_tls_ca
  tls_cert             = var.grafana_tls_cert
  tls_key              = var.grafana_tls_key
}
