resource "grafana_dashboard" "test" {
  config_json = <<EOD
{
  "uid": "report-dashboard",
  "title": "report-dashboard"
}
EOD
  message     = "inital commit."
}

resource "grafana_report" "test" {
  name       = "my report"
  recipients = ["some@email.com"]
  dashboards {
    uid = grafana_dashboard.test.uid
  }
  schedule {
    frequency = "hourly"
  }
}
