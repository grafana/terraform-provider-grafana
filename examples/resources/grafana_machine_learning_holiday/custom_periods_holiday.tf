resource "grafana_machine_learning_holiday" "custom_periods" {
  name        = "My custom periods holiday"
  description = "My Holiday"

  custom_periods {
    name       = "First of January"
    start_time = "2023-01-01T00:00:00Z"
    end_time   = "2023-01-02T00:00:00Z"
  }
  custom_periods {
    name       = "First of Feburary"
    start_time = "2023-02-01T00:00:00Z"
    end_time   = "2023-02-02T00:00:00Z"
  }
}

