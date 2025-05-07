resource "grafana_k6_project" "test" {
  name = "Terraform Test Project"
}

data "grafana_k6_project" "from_id" {
  depends_on = [
    grafana_k6_project.test
  ]
  id = grafana_k6_project.test.id
}


