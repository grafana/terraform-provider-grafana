resource "grafana_k6_project" "test_project_allowed_load_zones" {
  name = "Terraform Project Test Allowed Load Zones"
}

resource "grafana_k6_project_limits" "test_limits" {
  project_id              = grafana_k6_project.test_project_allowed_load_zones.id
  allowed_load_zones      = ["my-load-zone-1", "other-load-zone"]
}
