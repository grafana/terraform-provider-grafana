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
  name = "my report"
  dashboards {
    uid = grafana_dashboard.test.uid
  }
  recipients = ["some@email.com"]
  schedule {
    frequency         = "monthly"
    last_day_of_month = true
  }
}
