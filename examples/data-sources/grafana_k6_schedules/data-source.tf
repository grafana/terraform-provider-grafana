resource "grafana_k6_project" "schedules_project" {
  name = "Terraform Schedules Test Project"
}

resource "grafana_k6_load_test" "schedules_load_test" {
  project_id = grafana_k6_project.schedules_project.id
  name       = "Terraform Test Load Test for Schedules"
  script     = <<-EOT
    export default function() {
      console.log('Hello from k6 schedules test!');
    }
  EOT

  depends_on = [
    grafana_k6_project.schedules_project,
  ]
}

resource "grafana_k6_load_test" "schedules_load_test_2" {
  project_id = grafana_k6_project.schedules_project.id
  name       = "Terraform Test Load Test for Schedules (2)"
  script     = <<-EOT
    export default function() {
      console.log('Hello from k6 schedules test!');
    }
  EOT

  depends_on = [
    grafana_k6_project.schedules_project,
  ]
}


resource "grafana_k6_schedule" "test_schedule_1" {
  load_test_id = grafana_k6_load_test.schedules_load_test.id
  starts       = "2029-12-25T10:00:00Z"
  recurrence_rule {
    frequency = "MONTHLY"
    interval  = 15
    count     = 100
  }

  depends_on = [
    grafana_k6_load_test.schedules_load_test,
  ]
}

resource "grafana_k6_schedule" "test_schedule_2" {
  load_test_id = grafana_k6_load_test.schedules_load_test_2.id
  starts       = "2023-12-26T14:00:00Z"
  cron {
    schedule = "0 10 1 12 6"
    timezone = "UTC"
  }

  depends_on = [
    grafana_k6_load_test.schedules_load_test_2,
  ]
}

data "grafana_k6_schedules" "from_load_test_id" {

  depends_on = [
    grafana_k6_schedule.test_schedule_1,
    grafana_k6_schedule.test_schedule_2,
  ]
}
