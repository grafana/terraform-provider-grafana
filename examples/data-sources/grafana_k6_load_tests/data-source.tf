resource "grafana_k6_project" "load_test_project" {
  name = "Terraform Load Test Project"
}

resource "grafana_k6_load_test" "test_load_test" {
  project_id = grafana_k6_project.load_test_project.id
  name       = "Terraform Test Load Test"
  script     = <<-EOT
    export default function() {
      console.log('Hello from k6!');
    }
  EOT

  depends_on = [
    grafana_k6_project.load_test_project,
  ]
}

resource "grafana_k6_load_test" "test_load_test_2" {
  project_id = grafana_k6_project.load_test_project.id
  name       = "Terraform Test Load Test (2)"
  script     = <<-EOT
    export default function() {
      console.log('Hello from k6!');
    }
  EOT

  depends_on = [
    grafana_k6_load_test.test_load_test,
  ]
}

data "grafana_k6_load_tests" "from_project_id" {
  project_id = grafana_k6_project.load_test_project.id

  depends_on = [
    grafana_k6_load_test.test_load_test,
    grafana_k6_load_test.test_load_test_2
  ]
}

data "grafana_k6_load_tests" "filter_by_name" {
  name       = "Terraform Test Load Test (2)"
  project_id = grafana_k6_project.load_test_project.id

  depends_on = [
    grafana_k6_load_test.test_load_test_2,
  ]
}