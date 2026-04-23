resource "grafana_folder" "alertrule_folder" {
  title = "Alert Rule Folder"
}

resource "grafana_apps_rules_alertrule_v0alpha1" "example" {
  metadata {
    uid        = "example-alert-rule"
    folder_uid = grafana_folder.alertrule_folder.uid
  }
  spec {
    title = "Example Alert Rule"
    trigger {
      interval = "1m"
    }
    paused = true
    expressions = {
      "A" = {
        model = {
          datasource = {
            type = "prometheus"
            uid  = "ds_uid"
          }
          editorMode    = "code"
          expr          = "count(up{})"
          instant       = true
          intervalMs    = 1000
          legendFormat  = "__auto"
          maxDataPoints = 43200
          range         = false
          refId         = "A"
        }
        datasource_uid = "ds_uid"
        relative_time_range = {
          from = "600s"
          to   = "0s"
        }
        query_type = ""
        source     = true
      }
      "B" = {
        model = {
          conditions = [
            {
              evaluator = {
                params = [1]
                type   = "gt"
              }
              operator = {
                type = "and"
              }
              query = {
                params = ["C"]
              }
              reducer = {
                params = []
                type   = "last"
              }
              type = "query"
            }
          ]
          datasource = {
            type = "__expr__"
            uid  = "__expr__"
          }
          expression    = "A"
          intervalMs    = 1000
          maxDataPoints = 43200
          refId         = "C"
          type          = "threshold"
        }
        datasource_uid = "__expr__"
        query_type     = ""
        source         = false
      }
    }
    for = "5m"
    labels = {
      severity = "critical"
    }
    annotations = {
      runbook_url = "https://example.com"
    }
    no_data_state                   = "KeepLast"
    exec_err_state                  = "KeepLast"
    missing_series_evals_to_resolve = 5
    notification_settings {
      contact_point = "grafana-default-email"
    }
    panel_ref = {
      dashboard_uid = "dashboard123"
      panel_id      = "5"
    }
  }
}
