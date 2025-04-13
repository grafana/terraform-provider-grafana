resource "grafana_k6_project" "test_project_limits" {
  name = "Terraform Project Test Limits"
}

data "grafana_k6_project_limits" "from_id" {
  id = grafana_k6_project.test_project_limits.id
}

data "grafana_k6_project_limits" "from_project_id" {
  project_id = grafana_k6_project.test_project_limits.id
}