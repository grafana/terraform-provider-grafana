data "grafana_oncall_slack_channel" "example_slack_channel" {
  name = "example_slack_channel"
}
data "grafana_oncall_user_group" "example_user_group" {
  slack_handle = "example_slack_handle"
}


// ICal based schedule
resource "grafana_oncall_schedule" "example_schedule" {
  name      = "Example Ical Schadule"
  type      = "ical"
  ical_url  = "https://example.com/example_ical.ics"
  ical_url_overrides = "https://example.com/example_overrides_ical.ics"
  slack {
    channel_id = data.grafana_oncall_slack_channel.example_slack_channel.slack_id
    user_group_id = data.grafana_oncall_user_group.example_user_group.slack_id
  }
}

// Shift based schedule
resource "grafana_oncall_schedule" "example_schedule" {
  name      = "Example Calendar Schadule"
  type      = "calendar"
  time_zone = "America/New_York"
  shifts = [
  ]
  ical_url_overrides = "https://example.com/example_overrides_ical.ics"
}