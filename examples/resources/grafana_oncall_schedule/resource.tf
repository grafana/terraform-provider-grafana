data "grafana_oncall_slack_channel" "example_slack_channel" {
  name = "example_slack_channel"
}
data "grafana_oncall_user_group" "example_user_group" {
  slack_handle = "example_slack_handle"
}

data "grafana_team" "my_team" {
  name = "my team"
}

data "grafana_oncall_team" "my_team" {
  name = data.grafana_team.my_team.name
}

// ICal based schedule
resource "grafana_oncall_schedule" "example_schedule" {
  name               = "Example Ical Schadule"
  type               = "ical"
  ical_url_primary   = "https://example.com/example_ical.ics"
  ical_url_overrides = "https://example.com/example_overrides_ical.ics"

  // Optional: specify the team to which the schedule belongs
  team_id = data.grafana_oncall_team.my_team.id

  slack {
    channel_id    = data.grafana_oncall_slack_channel.example_slack_channel.slack_id
    user_group_id = data.grafana_oncall_user_group.example_user_group.slack_id
  }
}

// Shift based schedule
resource "grafana_oncall_schedule" "example_schedule" {
  name      = "Example Calendar Schadule"
  type      = "calendar"
  time_zone = "America/New_York"

  // Optional: specify the team to which the schedule belongs
  team_id = data.grafana_oncall_team.my_team.id

  shifts = [
  ]
  ical_url_overrides = "https://example.com/example_overrides_ical.ics"
}
