resource "grafana_dashboard" "test" {
  config_json = <<EOD
{
  "uid": "report-dashboard",
  "title": "report-dashboard"
}
EOD
  message     = "inital commit."
}

resource "grafana_dashboard" "test2" {
  config_json = <<EOD
{
  "uid": "report-dashboard-2",
  "title": "report-dashboard-2"
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
    start_time        = "2024-02-10T20:00:00"
    end_time          = "2024-02-15T15:00:00"
    timezone          = "America/New_York"
  }

  dashboards {
    uid = grafana_dashboard.test.uid
    time_range {
      from = "now-1h"
      to   = "now"
    }
    report_variables = {
      query0 = "a,b"
      query1 = "c,d"
    }
  }

  dashboards {
    uid = grafana_dashboard.test2.uid
  }
}
