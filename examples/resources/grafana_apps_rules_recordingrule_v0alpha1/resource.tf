resource "grafana_folder" "recordingrule_folder" {
  title = "Alert Rule Folder"
}


resource "grafana_apps_rules_recordingrule_v0alpha1" "example" {
  provider = grafana.local
  metadata {
    uid        = "example-recording-rule"
    folder_uid = grafana_folder.recordingrule_folder.uid
  }
  spec {
    title = "Example Recording Rule"
    trigger {
      interval = "1m"
    }
    paused = true
    expressions = {
      "A" = jsonencode({
        model = {
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
      })
    }
    target_datasource_uid = "target_ds_uid"
    metric                = "tf-metric"
    labels = {
      foo = "bar"
    }
  }
}
