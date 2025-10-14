resource "grafana_machine_learning_outlier_detector" "my_dbscan_outlier_detector" {
  name        = "My DBSCAN outlier detector"
  description = "My DBSCAN Outlier Detector"

  metric          = "tf_test_dbscan_job"
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
