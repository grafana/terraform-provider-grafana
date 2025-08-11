resource "grafana_asserts_log_drilldown_config" "example" {
  name   = "default"
  config = <<-EOT
    # Example log drilldown configuration
    # Structure depends on API schema; this is illustrative
    providers:
      - name: "loki"
        url: "https://logs.example.com"
        default: true
    mappings:
      - name: "kubernetes"
        matchers:
          - label: "job"
            value: "kubelet"
        query: "{job=\"kubelet\", namespace=\"default\"} |= \"error\""
  EOT
}
