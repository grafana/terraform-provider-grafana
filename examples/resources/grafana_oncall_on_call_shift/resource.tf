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
// Advanced example
////////

// Importing users
data "grafana_oncall_user" "all_users" {
  // Extract flat set of all users from the all teams
  for_each = toset(flatten([
    for team_name, username_list in local.teams : [
      username_list
    ]
  ]))
  username = each.key
}

// ON-CALL GROUPS / TEAMS
locals {
  teams = {
    emea = [
      "alfa@grafana.com",
      "bravo@grafana.com",
      "charlie@grafana.com",
      "echo@grafana.com",
      "delta@grafana.com",
      "foxtrot@grafana.com",
      "golf@grafana.com",
    ]
  }
  // oncall API operates with resources ID's, so we convert emails into ID's
  teams_map_of_user_id = { for team_name, username_list in local.teams : team_name => [
  for username in username_list : lookup(data.grafana_oncall_user.all_users, username).id] }
  users_map_by_id = { for username, oncall_user in data.grafana_oncall_user.all_users : oncall_user.id => oncall_user }
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
  // Run `terraform refresh` and `terraform output` to see the flattened list of users in the rotation
  rolling_users = [for k in flatten([
    local.teams_map_of_user_id.emea,
  ]) : [k]]
  start_rotation_from_user_index = 0

  // Optional: specify the team to which the on-call shift belongs
  team_id = data.grafana_oncall_team.my_team.id
}

output "emea_weekday__rolling_users" {
  value = [for k in flatten(grafana_oncall_on_call_shift.emea_weekday_shift.rolling_users) : lookup(local.users_map_by_id, k).username]
}
