resource "grafana_dashboard" "test" {
  config_json = <<EOD
{
  "title": "Dashboard for report",
  "uid": "report"
}
EOD
  message     = "inital commit."
}

resource "grafana_dashboard" "test2" {
  config_json = <<EOD
{
  "title": "Another dashboard for report",
  "uid": "report2"
}
EOD
  message     = "inital commit."
}

resource "grafana_report" "test" {
  name       = "multiple dashboards"
  recipients = ["some@email.com"]
  schedule {
    frequency         = "monthly"
    last_day_of_month = true
  }

  dashboards {
    uid = grafana_dashboard.test.uid
    time_range {
      from = "now-1h"
      to   = "now"
    }
  }

  dashboards {
    uid = grafana_dashboard.test2.uid
  }
}
