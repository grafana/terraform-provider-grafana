resource "grafana_asserts_threshold_rules" "test" {
  name  = "test-rule-request"
  scope = "request"
  rules = <<-EOT
    groups:
    - name: "custom-request-thresholds"
      rules:
      - alert: "HighErrorRatio"
        expr: "asserts:error:ratio:threshold > 0.1"
        for: "10m"
        labels:
          severity: "warning"
        annotations:
          summary: "High error ratio detected"
  EOT
}
