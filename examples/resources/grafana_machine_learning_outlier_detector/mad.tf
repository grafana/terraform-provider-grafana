resource "grafana_machine_learning_outlier_detector" "my_mad_outlier_detector" {
  name        = "My MAD outlier detector"
  description = "My MAD Outlier Detector"

  metric          = "tf_test_mad_job"
  datasource_type = "prometheus"
  datasource_uid  = "AbCd12345"
  query_params = {
    expr = "grafanacloud_grafana_instance_active_user_count"
  }
  interval = 300

  algorithm {
    name        = "mad"
    sensitivity = 0.7
  }
}
