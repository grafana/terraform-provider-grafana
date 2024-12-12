data "grafana_synthetic_monitoring_probes" "main" {}

resource "grafana_synthetic_monitoring_check" "browser" {
  job     = "Validate login"
  target  = "https://test.k6.io"
  enabled = true
  probes = [
    data.grafana_synthetic_monitoring_probes.main.probes.Paris,
  ]
  labels = {
    environment = "production"
  }
  settings {
    browser {
      // `browser_script.js` is a file in the same directory as this file and contains the
      // script to be executed.
      script = file("${path.module}/browser_script.js")
    }
  }
}
