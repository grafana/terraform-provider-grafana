resource "grafana_apps_notifications_templategroup_v1beta1" "example" {
  metadata {
    uid = "my-templates"
  }

  spec {
    title = "My Templates"

    content = <<-EOT
      {{ define "my_alert_summary" }}
      {{ .CommonLabels.alertname }}: {{ len .Alerts.Firing }} firing, {{ len .Alerts.Resolved }} resolved
      {{ end }}
    EOT
  }
}
