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
}

data "grafana_k6_load_test" "from_id" {
  id = grafana_k6_load_test.test_load_test.id
}
