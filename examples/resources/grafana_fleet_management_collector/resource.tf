resource "grafana_fleet_management_collector" "test" {
  id = "my_collector"
  remote_attributes = {
    "env"   = "PROD",
    "owner" = "TEAM-A"
  }
  enabled = true
}
