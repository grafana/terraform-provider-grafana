resource "grafana_synthetic_monitoring_check" "main" {
  job     = "Check Alert Test"
  target  = "https://grafana.com"
  enabled = true
  probes  = [1]
  labels  = {}
  settings {
    http {
      ip_version = "V4"
      method     = "GET"
    }
  }
}

resource "grafana_synthetic_monitoring_check_alerts" "main" {
  check_id = grafana_synthetic_monitoring_check.main.id
  alerts = [{
    name      = "ProbeFailedExecutionsTooHigh"
    threshold = 1
    period    = "15m"
    },
    {
      name      = "TLSTargetCertificateCloseToExpiring"
      threshold = 14
      period    = ""
  }]
} 