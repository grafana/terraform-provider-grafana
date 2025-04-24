resource "grafana_k6_project" "project" {
  name = "Terraform Test Project"
}

resource "grafana_k6_project" "project_2" {
  name = "Terraform Test Project"
}

data "grafana_k6_projects" "from_name" {
  name = "Terraform Test Project"

  depends_on = [
    grafana_k6_project.project,
    grafana_k6_project.project_2
  ]

}


