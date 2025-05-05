resource "grafana_k6_project" "load_test_project" {
  name = "Terraform Load Test Project"
}

resource "grafana_k6_load_test" "test_load_test_inline" {
  project_id = grafana_k6_project.load_test_project.id
  name       = "Terraform Test Load Test Inline"
  script     = <<-EOT
    export default function() {
      console.log('Hello from k6!');
    }
  EOT
}

resource "grafana_k6_load_test" "test_load_test_archive" {
  project_id  = grafana_k6_project.load_test_project.id
  name        = "Terraform Test Load Test Archive"
  script_file = "${path.module}/archive.tar"
}