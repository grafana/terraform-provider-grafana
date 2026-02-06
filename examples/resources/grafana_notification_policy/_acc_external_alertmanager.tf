# This example shows how to manage notification policies on an external alertmanager
# (e.g., grafanacloud-ngalertmanager) using the alertmanager_uid attribute.

resource "grafana_contact_point" "external_am_opsgenie" {
  alertmanager_uid = "grafanacloud-ngalertmanager"
  name             = "opsgenie"

  opsgenie {
    api_key = "your-api-key"
    url     = "https://api.eu.opsgenie.com/"
    message = "{{ .CommonAnnotations.summary }}"
  }
}

resource "grafana_contact_point" "external_am_oncall" {
  alertmanager_uid = "grafanacloud-ngalertmanager"
  name             = "grafana-oncall"

  webhook {
    url = "https://oncall.example.com/webhook"
  }
}

resource "grafana_notification_policy" "external_am" {
  alertmanager_uid = "grafanacloud-ngalertmanager"
  contact_point    = grafana_contact_point.external_am_opsgenie.name
  group_by         = ["cluster", "region", "alertname"]

  group_wait      = "10s"
  group_interval  = "1m"
  repeat_interval = "5m"

  # Send to Grafana OnCall first, then continue to default (OpsGenie)
  policy {
    contact_point = grafana_contact_point.external_am_oncall.name
    continue      = true
  }

  policy {
    contact_point = grafana_contact_point.external_am_opsgenie.name
  }
}
