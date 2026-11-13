data "grafana_oncall_users" "all" {}

// Map each user's email to their OnCall user ID.
// To look up users by Grafana login instead, key the map by user.username.
locals {
  oncall_user_ids_by_email = {
    for user in data.grafana_oncall_users.all.users : lower(user.email) => user.id
  }
}

// Replace the address with one of your own users: an address that matches nobody
// fails with "Invalid index".
output "alfa_oncall_user_id" {
  value = local.oncall_user_ids_by_email["alfa@example.com"]
}

// Replace the name with one of your own teams.
data "grafana_oncall_team" "my_team" {
  name = "my team"
}

// Read one team's members.
data "grafana_oncall_users" "my_team_members" {
  team_id = data.grafana_oncall_team.my_team.id
}

// Every user also lists the teams they belong to. For a configuration covering
// several teams, filter one read of all users instead of reading each team.
output "my_team_user_ids" {
  value = [
    for user in data.grafana_oncall_users.all.users : user.id
    if contains(user.teams, data.grafana_oncall_team.my_team.id)
  ]
}

// The blocks below create a shift and a schedule: they build a rotation from
// team membership rather than reading one.
//
// rotation_order decides the order people take call. Those listed in it keep
// their order relative to each other as the team changes. Everyone else follows
// them in user ID order, and a member who leaves the team drops out. An ID that
// is not on the team at all is ignored without an error.
//
// Leaving rotation_order empty is not a no-op: every member is then unlisted, so
// the whole rotation is ordered by user ID, which is arbitrary to a human.
locals {
  rotation_order = [] // OnCall user IDs, in the order they take call

  team_user_ids   = data.grafana_oncall_users.my_team_members.users[*].id
  ordered_users   = [for id in distinct(local.rotation_order) : id if contains(local.team_user_ids, id)]
  remaining_users = sort(setsubtract(local.team_user_ids, local.rotation_order))
  rotation_groups = [for id in concat(local.ordered_users, local.remaining_users) : [id]]
}

resource "grafana_oncall_on_call_shift" "my_team_rotation" {
  name          = "My team weekly rotation"
  type          = "rolling_users"
  start         = "2020-09-07T14:00:00"
  duration      = 60 * 60 * 24 * 7
  frequency     = "weekly"
  week_start    = "MO"
  interval      = 1
  time_zone     = "UTC"
  rolling_users = local.rotation_groups
  team_id       = data.grafana_oncall_team.my_team.id

  // A read that returns no members leaves rotation_groups empty, which empties
  // the rotation on apply. This stops the apply instead.
  lifecycle {
    precondition {
      condition     = length(local.rotation_groups) > 0
      error_message = "No members were found for this team, so applying would leave the rotation empty."
    }
  }
}

// A shift takes effect once a schedule uses it. Paging also needs an escalation
// chain with a notify_on_call_from_schedule step, and an integration with a route
// that uses that chain.
resource "grafana_oncall_schedule" "my_team" {
  name    = "My team schedule"
  type    = "calendar"
  shifts  = [grafana_oncall_on_call_shift.my_team_rotation.id]
  team_id = data.grafana_oncall_team.my_team.id
}
