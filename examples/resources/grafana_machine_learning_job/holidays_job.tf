resource "grafana_machine_learning_holiday" "test_holiday" {
  name = "Test Holiday"
  custom_periods {
    name       = "First of January"
    start_time = "2023-01-01T00:00:00Z"
    end_time   = "2023-01-02T00:00:00Z"
  }
}

resource "grafana_machine_learning_job" "test_job" {
  name            = "Test Job"
  metric          = "tf_test_job"
  datasource_type = "prometheus"
  datasource_id   = 10
  query_params = {
    expr = "grafanacloud_grafana_instance_active_user_count"
  }
  holidays = [
    grafana_machine_learning_holiday.test_holiday.id
  ]
}
