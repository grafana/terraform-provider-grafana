resource "grafana_dashboard" "test_bad_inputs" {
  config_json = jsonencode({
    id            = 12345,
    title         = "Production Overview",
    tags          = ["templated"],
    timezone      = "browser",
    schemaVersion = 16,
    version       = 0,
    refresh       = "25s"
  })
}

data "grafana_dashboard" "bad_from_uid_id" {
  uid          = grafana_dashboard.test_bad_inputs.id
  dashboard_id = grafana_dashboard.test_bad_inputs.dashboard_id
}
