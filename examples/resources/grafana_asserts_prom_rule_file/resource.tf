# Basic recording rule for latency metrics
resource "grafana_asserts_prom_rule_file" "latency_metrics" {
  name   = "custom-latency-metrics"
  active = true

  group {
    name     = "latency_recording_rules"
    interval = "30s"

    rule {
      record = "custom:latency:p95"
      expr   = "histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m]))"
      labels = {
        source   = "custom_instrumentation"
        severity = "info"
      }
    }

    rule {
      record = "custom:latency:p99"
      expr   = "histogram_quantile(0.99, rate(http_request_duration_seconds_bucket[5m]))"
      labels = {
        source   = "custom_instrumentation"
        severity = "info"
      }
    }
  }
}

# Alert rules for high latency
resource "grafana_asserts_prom_rule_file" "latency_alerts" {
  name   = "custom-latency-alerts"
  active = true

  group {
    name     = "latency_alerting"
    interval = "30s"

    rule {
      alert    = "HighLatency"
      expr     = "histogram_quantile(0.99, rate(http_request_duration_seconds_bucket[5m])) > 0.5"
      duration = "5m"
      labels = {
        severity = "warning"
        category = "Latency"
      }
      annotations = {
        summary     = "High latency detected"
        description = "P99 latency is above 500ms for 5 minutes"
      }
    }

    rule {
      alert    = "VeryHighLatency"
      expr     = "histogram_quantile(0.99, rate(http_request_duration_seconds_bucket[5m])) > 1.0"
      duration = "2m"
      labels = {
        severity = "critical"
        category = "Latency"
      }
      annotations = {
        summary     = "Very high latency detected"
        description = "P99 latency is above 1 second"
      }
    }
  }
}

# Comprehensive monitoring rules with multiple groups
resource "grafana_asserts_prom_rule_file" "comprehensive_monitoring" {
  name   = "custom-comprehensive-monitoring"
  active = true

  # Latency monitoring
  group {
    name     = "latency_monitoring"
    interval = "30s"

    rule {
      record = "custom:latency:p99"
      expr   = "histogram_quantile(0.99, rate(http_request_duration_seconds_bucket[5m]))"
      labels = {
        source = "custom"
      }
    }

    rule {
      alert    = "HighLatency"
      expr     = "custom:latency:p99 > 0.5"
      duration = "5m"
      labels = {
        severity = "warning"
      }
      annotations = {
        summary = "High latency detected"
      }
    }
  }

  # Error rate monitoring
  group {
    name     = "error_monitoring"
    interval = "1m"

    rule {
      record = "custom:error:rate"
      expr   = "rate(http_requests_total{status=~\"5..\"}[5m])"
      labels = {
        source = "custom"
      }
    }

    rule {
      alert    = "HighErrorRate"
      expr     = "custom:error:rate > 0.1"
      duration = "10m"
      labels = {
        severity = "critical"
        category = "Errors"
      }
      annotations = {
        summary     = "High error rate detected"
        description = "Error rate is above 10%"
      }
    }
  }

  # Throughput monitoring
  group {
    name     = "throughput_monitoring"
    interval = "1m"

    rule {
      record = "custom:throughput:total"
      expr   = "sum(rate(http_requests_total[5m]))"
      labels = {
        source = "custom"
      }
    }

    rule {
      alert    = "LowThroughput"
      expr     = "custom:throughput:total < 10"
      duration = "5m"
      labels = {
        severity = "warning"
        category = "Throughput"
      }
      annotations = {
        summary     = "Low throughput detected"
        description = "Request throughput is below 10 requests/second"
      }
    }
  }
}

# Rules with conditional enablement
resource "grafana_asserts_prom_rule_file" "conditional_rules" {
  name   = "custom-conditional-rules"
  active = true

  group {
    name     = "environment_specific_rules"
    interval = "30s"

    rule {
      alert    = "TestAlert"
      expr     = "up == 0"
      duration = "1m"
      labels = {
        severity = "info"
      }
      annotations = {
        summary = "Test alert that is disabled in production"
      }
      # This rule will be disabled in the production group
      disable_in_groups = ["production"]
    }

    rule {
      alert    = "CriticalAlert"
      expr     = "up == 0"
      duration = "30s"
      labels = {
        severity = "critical"
      }
      annotations = {
        summary = "Critical alert that fires in all environments"
      }
    }
  }
}

# Inactive rules (for staging/testing)
resource "grafana_asserts_prom_rule_file" "staging_rules" {
  name   = "custom-staging-rules"
  active = false # Rules file is inactive

  group {
    name     = "staging_tests"
    interval = "1m"

    rule {
      record = "staging:test:metric"
      expr   = "up"
      labels = {
        environment = "staging"
      }
    }
  }
}

# SLO-based alerting
resource "grafana_asserts_prom_rule_file" "slo_alerts" {
  name   = "custom-slo-alerts"
  active = true

  group {
    name     = "slo_monitoring"
    interval = "1m"

    rule {
      record = "custom:slo:availability"
      expr   = "sum(rate(http_requests_total{status!~\"5..\"}[5m])) / sum(rate(http_requests_total[5m]))"
      labels = {
        slo_type = "availability"
      }
    }

    rule {
      alert    = "SLOAvailabilityBreach"
      expr     = "custom:slo:availability < 0.995"
      duration = "5m"
      labels = {
        severity = "critical"
        category = "SLO"
      }
      annotations = {
        summary     = "SLO availability breach"
        description = "Availability is below 99.5% SLO target"
        runbook_url = "https://docs.example.com/runbooks/availability-breach"
      }
    }
  }
}

