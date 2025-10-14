resource "grafana_machine_learning_job" "test_alert_job" {
  name            = "Test Job"
  metric          = "tf_test_alert_job"
  datasource_type = "prometheus"
  datasource_uid  = "abcd12345"
  query_params = {
    expr = "grafanacloud_grafana_instance_active_user_count"
  }
}

resource "grafana_machine_learning_alert" "test_job_alert" {
  job_id            = grafana_machine_learning_job.test_alert_job.id
  title             = "Test Alert"
  anomaly_condition = "any"
  threshold         = ">0.8"
  window            = "15m"
  no_data_state     = "OK"
}
