resource "grafana_data_source" "foo" {
  type                = "prometheus"
  name                = "prometheus-ds-test"
  uid                 = "prometheus-ds-test-uid"
  url                 = "https://my-instance.com"
  basic_auth_enabled  = true
  basic_auth_username = "username"

  json_data_encoded = jsonencode({
    httpMethod        = "POST"
    prometheusType    = "Mimir"
    prometheusVersion = "2.4.0"
  })

  secure_json_data_encoded = jsonencode({
    basicAuthPassword = "password"
  })
}

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
  datasource_uid  = grafana_data_source.foo.uid
  query_params = {
    expr = "grafanacloud_grafana_instance_active_user_count"
  }
  holidays = [
    grafana_machine_learning_holiday.test_holiday.id
  ]
}
