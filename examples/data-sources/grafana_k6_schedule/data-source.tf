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
  recurrence_rule {
    frequency = "MONTHLY"
    interval  = 12
    count     = 100
  }

  depends_on = [
    grafana_k6_load_test.schedule_load_test,
  ]
}

data "grafana_k6_schedule" "from_id" {
  id = grafana_k6_schedule.test_schedule.id
}

output "complete_schedule_info" {
  description = "Complete schedule information"
  value = {
    id              = data.grafana_k6_schedule.from_id.id
    load_test_id    = data.grafana_k6_schedule.from_id.load_test_id
    starts          = data.grafana_k6_schedule.from_id.starts
    deactivated     = data.grafana_k6_schedule.from_id.deactivated
    next_run        = data.grafana_k6_schedule.from_id.next_run
    created_by      = data.grafana_k6_schedule.from_id.created_by
    recurrence_rule = data.grafana_k6_schedule.from_id.recurrence_rule
  }
}
