resource "grafana_folder" "my_folder" {
  title = "My Alert Folder"
}

resource "grafana_alerting_rule" "my_rule" {
  config_json = jsonencode({
    title        = "High Error Rate Alert"
    uid          = "my-error-rate-alert"
    condition    = "B"
    folderUID    = grafana_folder.my_folder.uid
    ruleGroup    = "Error Rate Alerts"
    noDataState  = "NoData"
    execErrState = "Error"
    for          = "5m"
    data = [
      {
        refId         = "A"
        datasourceUid = "prometheus"
        model = {
          expr  = "rate(http_requests_total{status=~\"5..\"}[5m])"
          refId = "A"
        }
        relativeTimeRange = {
          from = 600
        }
      },
      {
        refId         = "B"
        datasourceUid = "__expr__"
        model = {
          type       = "threshold"
          expression = "A"
          conditions = [{
            evaluator = {
              type   = "gt"
              params = [0.01]
            }
          }]
        }
      }
    ]
  })
}
