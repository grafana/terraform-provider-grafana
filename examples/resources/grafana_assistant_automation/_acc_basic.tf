resource "grafana_assistant_automation" "test" {
  name        = "tf-acc-test-automation"
  description = "Terraform acceptance test automation."
  prompt      = "Summarize the alerts that fired in the last 24 hours."
  scope       = "tenant"

  # Kept disabled so the acceptance test never triggers a scheduled run.
  enabled           = false
  schedule_cron     = "0 9 * * *"
  schedule_timezone = "UTC"
}
