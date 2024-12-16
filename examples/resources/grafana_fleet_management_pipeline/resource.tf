resource "grafana_fleet_management_pipeline" "test" {
  name     = "my_pipeline"
  contents = file("config.alloy")
  matchers = [
    "collector.os=\"linux\"",
    "owner=\"TEAM-A\""
  ]
  enabled = true
}
