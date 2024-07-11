resource "grafana_machine_learning_outlier_detector" "test_alert_outlier_detector" {
  name = "Test Outlier"

  metric          = "tf_test_alert_outlier"
  datasource_type = "prometheus"
  datasource_uid  = "AbCd12345"
  query_params = {
    expr = "grafanacloud_grafana_instance_active_user_count"
  }
  interval = 300

  algorithm {
    name        = "dbscan"
    sensitivity = 0.5
    config {
      epsilon = 1.0
    }
  }
}

resource "grafana_machine_learning_alert" "test_outlier_alert" {
  outlier_id = grafana_machine_learning_outlier_detector.test_alert_outlier_detector.id
  title      = "Test Alert"
  window     = "1h"
}
