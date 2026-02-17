# This example shows how to manage contact points on an external alertmanager
# (e.g., grafanacloud-ngalertmanager) using the alertmanager_uid attribute.
#
# When using alertmanager_uid with a native (non-Grafana-managed) alertmanager,
# the provider automatically converts notifier fields to native Alertmanager format.

resource "grafana_contact_point" "external_am_opsgenie" {
  alertmanager_uid = "grafanacloud-ngalertmanager"
  name             = "opsgenie"

  opsgenie {
    api_key = "your-api-key"
    url     = "https://api.eu.opsgenie.com/"
    message = "{{ .CommonAnnotations.summary }}"
    settings = {
      # For native alertmanager, og_priority is automatically mapped to "priority"
      og_priority = "P3"
      tags        = "env={{ .CommonLabels.env }}"
    }
  }
}

resource "grafana_contact_point" "external_am_webhook" {
  alertmanager_uid = "grafanacloud-ngalertmanager"
  name             = "webhook"

  webhook {
    url = "https://example.com/webhook"
  }
}
