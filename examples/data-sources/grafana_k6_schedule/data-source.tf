resource "grafana_k6_project" "schedule_project" {
  name = "Terraform Schedule Test Project"
}

resource "grafana_k6_load_test" "schedule_load_test" {
  project_id = grafana_k6_project.schedule_project.id
  name       = "Terraform Test Load Test for Schedule"
  script     = <<-EOT
    export default function() {
      console.log('Hello from k6 schedule test!');
    }
  EOT

  depends_on = [
    grafana_k6_project.schedule_project,
  ]
}

resource "grafana_k6_schedule" "test_schedule" {
  load_test_id = grafana_k6_load_test.schedule_load_test.id
  starts       = "2024-12-25T10:00:00Z"
  frequency    = "MONTHLY"
  interval     = 12
  occurrences  = 100

  depends_on = [
    grafana_k6_load_test.schedule_load_test,
  ]
}

data "grafana_k6_schedule" "from_id" {
  id = grafana_k6_schedule.test_schedule.id
}
