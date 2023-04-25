resource "grafana_machine_learning_job" "test_job" {
  name            = "Test Job"
  metric          = "tf_test_job"
  datasource_type = "prometheus"
  datasource_uid  = "grafanacloud-usage"
  query_params = {
    expr = "grafanacloud_grafana_instance_active_user_count"
  }
  hyper_params = {
    daily_seasonality  = 15
    weekly_seasonality = 10
  }
  custom_labels = {
    example_label = "example_value"
  }
}
