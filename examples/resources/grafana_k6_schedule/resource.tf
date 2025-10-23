resource "grafana_k6_project" "schedule_project" {
  name = "Terraform Schedule Resource Project"
}

resource "grafana_k6_load_test" "scheduled_test" {
  project_id = grafana_k6_project.schedule_project.id
  name       = "Terraform Scheduled Resource Test"
  script     = <<-EOT
    export default function() {
      console.log('Hello from scheduled k6 test!');
    }
  EOT

  depends_on = [
    grafana_k6_project.schedule_project,
  ]
}

resource "grafana_k6_schedule" "cron_monthly" {
  load_test_id = grafana_k6_load_test.scheduled_test.id
  starts       = "2024-12-25T10:00:00Z"
  cron {
    schedule = "0 10 1 * *"
    timezone = "UTC"
  }
}


resource "grafana_k6_schedule" "daily" {
  load_test_id = grafana_k6_load_test.scheduled_test.id
  starts       = "2024-12-25T10:00:00Z"
  recurrence_rule {
    frequency = "DAILY"
    interval  = 1
  }
}

resource "grafana_k6_schedule" "weekly" {
  load_test_id = grafana_k6_load_test.scheduled_test.id
  starts       = "2024-12-25T09:00:00Z"
  recurrence_rule {
    frequency = "WEEKLY"
    interval  = 1
    byday     = ["MO", "WE", "FR"]
  }
}

# Example with YEARLY frequency and count
resource "grafana_k6_schedule" "yearly" {
  load_test_id = grafana_k6_load_test.scheduled_test.id
  starts       = "2024-01-01T12:00:00Z"
  recurrence_rule {
    frequency = "YEARLY" # Valid enum value
    interval  = 1
    count     = 5 # Run 5 times total
  }
}

# One-time schedule without recurrence
resource "grafana_k6_schedule" "one_time" {
  load_test_id = grafana_k6_load_test.scheduled_test.id
  starts       = "2024-12-25T15:00:00Z"
  # No recurrence_rule means it runs only once
}
