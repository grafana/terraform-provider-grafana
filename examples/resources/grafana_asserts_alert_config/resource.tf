resource "grafana_asserts_notification_alerts_config" "high_error_rate" {
  name = "HighErrorRate"

  match_labels = {
    service = "api-service"
    env     = "production"
  }

  alert_labels = {
    severity = "critical"
    team     = "platform"
  }

  duration = "5m"
  silenced = false
}

resource "grafana_asserts_notification_alerts_config" "slow_response_time" {
  name = "SlowResponseTime"

  match_labels = {
    service = "web-frontend"
    env     = "production"
  }

  alert_labels = {
    severity = "warning"
    team     = "frontend"
  }

  duration = "10m"
  silenced = false
} 