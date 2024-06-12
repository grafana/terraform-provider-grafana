data "grafana_synthetic_monitoring_probes" "main" {}

data "local_file" "script" {
  // `script.js` is a file in the same directory as this file and contains the
  // script to be executed.
  //
  // The content of the file will be read and stored in the content attribute.
  // The content attribute is a string, so it can be used directly in the
  // settings block below.
  filename = "${path.module}/script.js"
}

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
      script = data.local_file.script.content
    }
  }
}
