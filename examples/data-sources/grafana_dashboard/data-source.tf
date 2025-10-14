resource "grafana_dashboard" "test" {
  config_json = jsonencode({
    id            = 12345,
    uid           = "test-ds-dashboard-uid"
    title         = "Production Overview",
    tags          = ["templated"],
    timezone      = "browser",
    schemaVersion = 16,
    version       = 0,
    refresh       = "25s"
  })
}

data "grafana_dashboard" "from_uid" {
  depends_on = [
    grafana_dashboard.test
  ]
  uid = "test-ds-dashboard-uid"
}
