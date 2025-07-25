resource "grafana_k6_project" "schedule_project" {
  name = "Terraform Schedule Project"
}

resource "grafana_k6_load_test" "scheduled_test" {
  project_id = grafana_k6_project.schedule_project.id
  name       = "Terraform Scheduled Test"
  script     = <<-EOT
    export default function() {
      console.log('Hello from scheduled k6 test!');
    }
  EOT
}

resource "grafana_k6_load_test" "scheduled_test_2" {
  project_id = grafana_k6_project.schedule_project.id
  name       = "Terraform Scheduled Test 2"
  script     = <<-EOT
    export default function() {
      console.log('Hello from scheduled k6 test 2!');
    }
  EOT
}

resource "grafana_k6_load_test" "scheduled_test_3" {
  project_id = grafana_k6_project.schedule_project.id
  name       = "Terraform Scheduled Test 3"
  script     = <<-EOT
    export default function() {
      console.log('Hello from scheduled k6 test!');
    }
  EOT
}

# Basic schedule - runs daily at 9 AM UTC
resource "grafana_k6_schedule" "daily_schedule" {
  load_test_id = grafana_k6_load_test.scheduled_test.id
  starts       = "2024-01-01T09:00:00Z"
  frequency    = "DAILY"
}

# Advanced schedule - runs every 2 hours with occurrences limit
resource "grafana_k6_schedule" "hourly_schedule" {
  load_test_id = grafana_k6_load_test.scheduled_test_2.id
  starts       = "2024-01-01T08:00:00Z"
  frequency    = "HOURLY"
  interval     = 2
  occurrences  = 50
}

# Weekly schedule with end date
resource "grafana_k6_schedule" "weekly_schedule" {
  load_test_id = grafana_k6_load_test.scheduled_test_3.id
  starts       = "2024-01-01T14:30:00Z"
  frequency    = "WEEKLY"
  until        = "2024-12-31T23:59:59Z"
}
