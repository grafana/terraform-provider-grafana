resource "grafana_asserts_threshold_rules" "test" {
  name  = "test-rule-resource"
  scope = "resource"
  rules = <<-EOT
    groups:
    - name: "custom-resource-thresholds"
      rules:
      - alert: "CPUSaturationCritical"
        expr: "asserts:resource:threshold{type='cpu:usage'} > 0.8"
        for: "15m"
        labels:
          severity: "critical"
        annotations:
          summary: "High CPU saturation"
  EOT
}
