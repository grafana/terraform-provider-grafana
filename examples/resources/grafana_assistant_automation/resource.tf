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

  # Default: runs use the creator's permissions.
  # service_account requires tenant scope and organization Admin permission.
  execution_mode = "user"
  labels = {
    application = "ecommerce"
    managed_by  = "terraform"
  }

  # Multiple expressions can be separated by semicolons.
  # Combined firings must be at least 15 minutes apart.
  schedule_cron     = "0 9 * * *"
  schedule_timezone = "Europe/Paris"
  # Optional, if the backend enables bounded schedules. Omit to clear.
  # schedule_ends_at = "2030-01-01T00:00:00Z"

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

# Event-only automation. Replace the watcher UUID with an existing Watcher.
# The backend must enable Watcher-to-Automation triggers. No events run while disabled.
resource "grafana_assistant_automation" "on_critical_finding" {
  name           = "Investigate critical Watcher findings"
  prompt         = "Investigate the critical finding supplied by the Watcher event."
  scope          = "tenant"
  execution_mode = "service_account"
  enabled        = false

  # Trigger keys are stable UUIDs. Do not call uuid() here: it changes each plan.
  # Keep the key when editing a trigger; remove its entry to unsubscribe.
  triggers = {
    "c971a626-703b-4472-8eb0-e92aaf705af9" = {
      type = "watcher_finding"
      config = jsonencode({
        watcherId = "52fcc709-4ab5-45c8-8f79-a1c1d6abef17"
      })
    }
  }
}

# A manual-only automation needs neither schedule nor timezone.
resource "grafana_assistant_automation" "manual_report" {
  name    = "On-demand incident report"
  prompt  = "Summarize the incidents from the past 24 hours."
  enabled = false
}
