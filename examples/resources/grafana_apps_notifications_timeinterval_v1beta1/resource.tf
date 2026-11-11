resource "grafana_apps_notifications_timeinterval_v1beta1" "example" {
  metadata {
    uid = "business-hours"
  }

  spec {
    name = "business-hours"

    time_intervals {
      weekdays = ["monday:friday"]

      times = [
        {
          start_time = "09:00"
          end_time   = "17:00"
        }
      ]
    }
  }
}
