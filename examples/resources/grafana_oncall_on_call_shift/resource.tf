data "grafana_oncall_user" "alex" {
  username = "alex"
}

data "grafana_team" "my_team" {
  name = "my team"
}

data "grafana_oncall_team" "my_team" {
  name = data.grafana_team.my_team.name
}

resource "grafana_oncall_on_call_shift" "example_shift" {
  name       = "Example Shift"
  type       = "recurrent_event"
  start      = "2020-09-07T14:00:00"
  duration   = 60 * 30
  frequency  = "weekly"
  interval   = 2
  by_day     = ["MO", "FR"]
  week_start = "MO"
  users = [
    data.grafana_oncall_user.alex.id
  ]
  time_zone = "UTC"

  // Optional: specify the team to which the on-call shift belongs
  team_id = data.grafana_oncall_team.my_team.id
}

////////
// Advanced example: a rotation built from a list of emails
////////

// Reads every OnCall user once, instead of one data source per person
data "grafana_oncall_users" "all" {}

locals {
  // The people in the rotation, in order. Each person takes one turn.
  emea_rotation = [
    "alfa@example.com",
    "bravo@example.com",
    "charlie@example.com",
  ]

  // To list people by Grafana login instead, key this map by user.username
  oncall_user_ids_by_email = {
    for user in data.grafana_oncall_users.all.users : lower(user.email) => user.id
  }
  emea_missing_emails = [
    for email in local.emea_rotation : email
    if !contains(keys(local.oncall_user_ids_by_email), lower(email))
  ]
}

// A 12 hour shift on week days with the on-call person rotating weekly.
resource "grafana_oncall_on_call_shift" "emea_weekday_shift" {
  name       = "EMEA Weekday Shift"
  type       = "rolling_users"
  start      = "2022-02-28T03:00:00"
  duration   = 60 * 60 * 12 // 12 hours
  frequency  = "weekly"
  interval   = 1
  by_day     = ["MO", "TU", "WE", "TH", "FR"]
  week_start = "MO"
  time_zone  = "UTC"

  rolling_users = [
    for email in local.emea_rotation : [local.oncall_user_ids_by_email[lower(email)]]
  ]
  start_rotation_from_user_index = 0

  // Optional: specify the team to which the on-call shift belongs
  team_id = data.grafana_oncall_team.my_team.id

  lifecycle {
    precondition {
      condition     = length(local.emea_missing_emails) == 0
      error_message = "No OnCall user has these emails: ${join(", ", local.emea_missing_emails)}"
    }
  }
}
