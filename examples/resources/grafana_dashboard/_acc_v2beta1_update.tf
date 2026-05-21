resource "grafana_dashboard" "v2beta1" {
  config_json = <<EOD
{
  "apiVersion": "dashboard.grafana.app/v2beta1",
  "metadata": {
    "name": "abc123"
  },
  "kind": "Dashboard",
  "spec": {
    "title": "DashboardV2beta1 Updated",
    "elements": {},
    "annotations": null,
    "cursorSync": "",
    "links": null,
    "preload": false,
    "tags": null,
    "variables": null,
    "layout": {
      "kind": "GridLayout",
      "spec": {
        "items": null
      }
    },
    "timeSettings": {
      "autoRefresh": "",
      "autoRefreshIntervals": null,
      "fiscalYearStartMonth": 0,
      "from": "",
      "hideTimepicker": false,
      "to": ""
    }
  }
}
EOD
  message     = "dashboard v2beta1 updated"
}
