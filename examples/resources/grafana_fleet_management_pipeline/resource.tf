resource "grafana_fleet_management_pipeline" "test" {
  name     = "my_pipeline"
  contents = file("config.alloy")
  matchers = [
    "collector.os=~\".*\"",
    "env=\"PROD\""
  ]
  enabled = true

  # Pipelines are always labeled as Terraform-managed in Fleet Management.
  # Optional namespace for that source (default "default"), e.g. terraform.workspace:
  # terraform_source_namespace = terraform.workspace
}
