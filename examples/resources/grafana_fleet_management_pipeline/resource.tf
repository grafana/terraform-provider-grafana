resource "grafana_fleet_management_pipeline" "test" {
  name     = "my_pipeline"
  contents = file("config.alloy")
  matchers = [
    "collector.os=~\".*\"",
    "env=\"PROD\""
  ]
  enabled = true

  # Pipelines are labeled as Terraform-managed in Fleet Management by default.
  # Optional namespace for that source (default "default"), e.g. terraform.workspace:
  # terraform_source_namespace = terraform.workspace
  # Set disable_provenance to true to allow editing outside Terraform.
  # disable_provenance = true
}
