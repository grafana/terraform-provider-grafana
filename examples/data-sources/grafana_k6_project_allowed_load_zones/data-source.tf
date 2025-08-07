resource "grafana_k6_project" "test_project_allowed_load_zones" {
  name = "Terraform Project Test Allowed Load Zones"
}

data "grafana_k6_project_allowed_load_zones" "from_project_id" {
  project_id = grafana_k6_project.test_project_allowed_load_zones.id
}