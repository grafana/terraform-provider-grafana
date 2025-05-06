resource "grafana_k6_project" "test_project_limits" {
  name = "Terraform Project Test Limits"
}

resource "grafana_k6_project_limits" "test_limits" {
  project_id              = grafana_k6_project.test_project_limits.id
  vuh_max_per_month       = 1000
  vu_max_per_test         = 100
  vu_browser_max_per_test = 10
  duration_max_per_test   = 3600
}