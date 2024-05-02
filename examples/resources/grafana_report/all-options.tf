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
  // Required attributes
  name = "my report updated"
  dashboards {
    uid = grafana_dashboard.test.uid
    time_range {
      from = "now-1h"
      to   = "now"

    }
  }
  recipients = ["some@email.com", "some2@email.com"]
  schedule {
    frequency     = "daily"
    workdays_only = true
    start_time    = "2020-01-01T00:00:00-07:00"
    end_time      = "2020-01-15T16:00:00+07:30"
  }

  // Optional attributes
  orientation            = "portrait"
  layout                 = "simple"
  include_dashboard_link = false
  include_table_csv      = true
  formats                = ["csv", "image", "pdf"]
}
