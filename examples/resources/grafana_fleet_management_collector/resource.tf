resource "grafana_fleet_management_collector" "test" {
  id = "my_collector"
  attribute_overrides = {
    "env"   = "PROD",
    "owner" = "TEAM-A"
  }
  enabled = true
}
