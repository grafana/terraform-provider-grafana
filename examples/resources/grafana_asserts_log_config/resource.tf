# Production environment with comprehensive logging
resource "grafana_asserts_log_config" "production" {
  name   = "production"
  config = <<-EOT
    name: production
    logConfig:
      enabled: true
      retention: "30d"
      maxLogSize: "1GB"
      compression: true
      filters:
        - level: "ERROR"
        - level: "WARN"
        - service: "api"
        - service: "web"
      sampling:
        rate: 0.1
        maxTracesPerSecond: 100
  EOT
}

# Development environment with minimal configuration
resource "grafana_asserts_log_config" "development" {
  name   = "development"
  config = <<-EOT
    name: development
    logConfig:
      enabled: true
      retention: "7d"
      maxLogSize: "100MB"
      compression: false
  EOT
}

# Staging environment with moderate settings
resource "grafana_asserts_log_config" "staging" {
  name   = "staging"
  config = <<-EOT
    name: staging
    logConfig:
      enabled: true
      retention: "14d"
      maxLogSize: "500MB"
      compression: true
      filters:
        - level: "ERROR"
        - level: "WARN"
        - level: "INFO"
  EOT
}

# Minimal configuration for testing
resource "grafana_asserts_log_config" "test" {
  name   = "test"
  config = <<-EOT
    name: test
    logConfig:
      enabled: true
  EOT
}