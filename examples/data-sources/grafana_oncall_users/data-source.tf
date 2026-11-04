data "grafana_oncall_users" "all" {}

// Map each user's email to their OnCall user ID.
// To look up users by Grafana login instead, key the map by user.username.
locals {
  oncall_user_ids_by_email = {
    for user in data.grafana_oncall_users.all.users : lower(user.email) => user.id
  }
}

output "alfa_oncall_user_id" {
  value = local.oncall_user_ids_by_email["alfa@example.com"]
}
