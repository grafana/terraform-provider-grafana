data "grafana_synthetic_monitoring_probes" "main" {}

resource "grafana_synthetic_monitoring_check" "scripted" {
  job     = "Validate homepage"
  target  = "https://grafana.com/"
  enabled = true
  probes = [
    data.grafana_synthetic_monitoring_probes.main.probes.Paris,
  ]
  labels = {
    environment = "production"
  }
  settings {
    scripted {
      // `script.js` is a file in the same directory as this file and contains the
      // script to be executed.
      script = file("${path.module}/script.js")
    }
  }
}
