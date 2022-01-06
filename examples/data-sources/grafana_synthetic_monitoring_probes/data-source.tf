data "grafana_synthetic_monitoring_probes" "main" {}

data "grafana_synthetic_monitoring_probes" "not_deprecated" {
  filter_deprecated = true
}
