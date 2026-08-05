resource "grafana_assistant_automation" "daily_incident_report" {
  name        = "Daily ecommerce incident report"
  description = "Every morning, summarize the alerts that fired for the ecommerce app."

  prompt = <<-EOT
    Produce a daily report on the incidents for the main alerts that fired in
    the last 24 hours for the ecommerce app in the ecommerce-prod namespace.

    For each alert: how long it fired, which services were affected, and
    whether it correlates with a deploy. Close with the single most important
    thing to look at today.
  EOT

  # Shared with everybody in the org, rather than private to the creator.
  scope   = "tenant"
  enabled = true

  # Standard 5-field cron. Schedules must be at least 15 minutes apart.
  schedule_cron     = "0 9 * * *"
  schedule_timezone = "Europe/Paris"

  notifications {
    slack {
      enabled   = true
      notify_on = ["completed", "failed"]

      target {
        type       = "channel"
        channel_id = "C0123456789"
      }
    }
  }
}
