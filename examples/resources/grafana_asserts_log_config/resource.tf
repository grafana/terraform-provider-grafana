resource "grafana_asserts_log_config" "production" {
  name = "production"
  
  config = <<-EOT
    name: "production"
    envsForLog:
      - "production"
      - "staging"
    sitesForLog:
      - "us-east-1"
      - "us-west-2"
    logConfig:
      tool: "loki"
      url: "https://logs.example.com"
      dateFormat: "RFC3339"
      correlationLabels: "trace_id,span_id"
      defaultSearchText: "error"
      errorFilter: "level=error"
      columns:
        - "timestamp"
        - "level"
        - "message"
      index: "logs-*"
      interval: "1h"
      query:
        job: "app"
        level: "error"
      sort:
        - "timestamp desc"
      httpResponseCodeField: "status_code"
      orgId: "1"
      dataSource: "loki"
    defaultConfig: false
  EOT
}

resource "grafana_asserts_log_config" "development" {
  name = "development"
  
  config = <<-EOT
    name: "development"
    envsForLog:
      - "development"
      - "testing"
    sitesForLog:
      - "us-east-1"
    logConfig:
      tool: "elasticsearch"
      url: "https://elastic-dev.example.com"
      dateFormat: "ISO8601"
      correlationLabels: "trace_id,span_id,request_id"
      defaultSearchText: "warning"
      errorFilter: "level=error OR level=warning"
      columns:
        - "timestamp"
        - "level"
        - "message"
        - "service"
      index: "dev-logs-*"
      interval: "30m"
      query:
        job: "app"
        level: "error"
        service: "api"
      sort:
        - "timestamp desc"
        - "level asc"
      httpResponseCodeField: "status_code"
      orgId: "1"
      dataSource: "elasticsearch"
    defaultConfig: true
  EOT
}

resource "grafana_asserts_log_config" "minimal" {
  name = "minimal"
  
  config = <<-EOT
    name: "minimal"
    logConfig:
      tool: "loki"
      url: "https://logs-minimal.example.com"
    defaultConfig: false
  EOT
}
