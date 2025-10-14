import {
  id = "12345"
  to = grafana_synthetic_monitoring_check._12345
}

resource "grafana_synthetic_monitoring_check" "_12345" {
  alert_sensitivity  = "none"
  basic_metrics_only = true
  enabled            = false
  frequency          = 60000
  job                = "testname"
  labels = {
    foo = "bar"
  }
  probes  = [7]
  target  = "https://grafana.com"
  timeout = 3000
  settings {
    http {
      fail_if_not_ssl     = false
      fail_if_ssl         = false
      ip_version          = "V4"
      method              = "GET"
      no_follow_redirects = false
    }
  }
}
