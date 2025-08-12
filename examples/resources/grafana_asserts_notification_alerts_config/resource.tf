# Basic alert configuration with silencing
resource "grafana_asserts_notification_alerts_config" "prometheus_remote_storage_failures" {
  name = "PrometheusRemoteStorageFailures"

  match_labels = {
    alertname   = "PrometheusRemoteStorageFailures"
    alertgroup  = "prometheus.alerts"
    asserts_env = "prod"
  }

  silenced = true
}

# High severity alert with specific job and context matching
resource "grafana_asserts_notification_alerts_config" "error_buildup_notify" {
  name = "ErrorBuildupNotify"

  match_labels = {
    alertname               = "ErrorBuildup"
    job                     = "acai"
    asserts_request_type    = "inbound"
    asserts_request_context = "/auth"
  }

  silenced = false
}

# Alert with additional labels and custom duration
resource "grafana_asserts_notification_alerts_config" "payment_test_alert" {
  name = "PaymentTestAlert"

  match_labels = {
    alertname         = "PaymentTestAlert"
    additional_labels = "asserts_severity=~\"critical\""
    alertgroup        = "alex-k8s-integration-test.alerts"
  }

  alert_labels = {
    testing = "onetwothree"
  }

  duration = "5m"
  silenced = false
}

# Latency alert for shipping service
resource "grafana_asserts_notification_alerts_config" "high_shipping_latency" {
  name = "high shipping latency"

  match_labels = {
    alertname            = "LatencyP99ErrorBuildup"
    job                  = "shipping"
    asserts_request_type = "inbound"
  }

  silenced = false
}

# CPU throttling alert with warning severity
resource "grafana_asserts_notification_alerts_config" "cpu_throttling_sustained" {
  name = "CPUThrottlingSustained"

  match_labels = {
    alertname         = "CPUThrottlingSustained"
    additional_labels = "asserts_severity=~\"warning\""
  }

  silenced = true
}

# Ingress error rate alert
resource "grafana_asserts_notification_alerts_config" "ingress_error" {
  name = "ingress error"

  match_labels = {
    alertname            = "ErrorRatioBreach"
    job                  = "ingress-nginx-controller-metrics"
    asserts_request_type = "inbound"
  }

  silenced = false
}

# MySQL Galera cluster alert
resource "grafana_asserts_notification_alerts_config" "mysql_galera_not_ready" {
  name = "MySQLGaleraNotReady"

  match_labels = {
    alertname = "MySQLGaleraNotReady"
  }

  silenced = false
}
