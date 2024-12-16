resource "grafana_fleet_management_pipeline" "test" {
  name     = "my_pipeline"
  contents = file("config.alloy")
  matchers = [
    "collector.os=~\".*\"",
    "env=\"PROD\""
  ]
  enabled = true
}
