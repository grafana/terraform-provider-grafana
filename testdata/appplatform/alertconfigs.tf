resource "grafana_apps_asserts_alertconfig_v2alpha1" "test_alert_config_one" {
  metadata {
    uid        = "test_alert_config_one"
    folder_uid = grafana_folder.test_folder_one.uid
  }

  spec {
    match_labels = {
      alertname = "HighCPUUsage"
    }
    alert_labels = {
      severity = "warning"
      team     = "platform"
    }
    duration = "5m"
    silenced = false
  }

  options {
    overwrite = true
  }
}

resource "grafana_apps_asserts_alertconfig_v2alpha1" "test_alert_config_two" {
  metadata {
    uid        = "test_alert_config_two"
    folder_uid = grafana_folder.test_folder_two.uid
  }

  spec {
    match_labels = {
      asserts_slo_name = "api-latency"
    }
    alert_labels = {
      severity = "critical"
      team     = "backend"
    }
    duration = "2m"
    silenced = true
  }

  options {
    overwrite = true
  }
} 