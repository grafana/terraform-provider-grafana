resource "grafana_apps_alertenrichment_alertenrichment_v1beta1" "example" {
  metadata = {
    uid = "alert-enrichment-example"
  }

  spec {
    title       = "Critical Alert Enrichment Pipeline"
    description = "Enriches critical production alerts with team information"

    # Apply only to specific alert rules
    alert_rule_uids = ["high-cpu-alert", "disk-space-alert"]

    # Apply only to specific receivers
    receivers = ["critical-alerts", "alerting-team"]

    # Match alerts with specific labels
    label_matchers = [
      {
        type  = "="
        name  = "severity"
        value = "critical"
      },
      {
        type  = "=~" # Regex match
        name  = "environment"
        value = "prod.*"
      },
      {
        type  = "!="
        name  = "team"
        value = "test"
      }
    ]

    # Match alerts with specific annotations
    annotation_matchers = [
      {
        type  = "!~" # Regex not match
        name  = "runbook_url"
        value = "^http://grafana.com$"
      }
    ]

    assign_step {
      annotations = {
        enrichment_team   = "alerting-team"
        runbook_url       = "https://runbooks.grafana.com/critical-alerts"
        contact_slack     = "#alerts-critical"
        incident_severity = "high"
      }
      timeout = "30s"
    }
  }
}
