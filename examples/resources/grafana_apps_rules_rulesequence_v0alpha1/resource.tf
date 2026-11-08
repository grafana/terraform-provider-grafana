resource "grafana_folder" "rulesequence_folder" {
  title = "Rule Sequence Folder"
}

resource "grafana_apps_rules_recordingrule_v0alpha1" "example" {
  metadata {
    uid        = "example-seq-recording-rule"
    folder_uid = grafana_folder.rulesequence_folder.uid
  }
  spec {
    title = "Example Sequence Recording Rule"
    trigger {
      interval = "1m"
    }
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
    metric                = "tf-seq-metric"
  }
}

resource "grafana_apps_rules_alertrule_v0alpha1" "example" {
  metadata {
    uid        = "example-seq-alert-rule"
    folder_uid = grafana_folder.rulesequence_folder.uid
  }
  spec {
    title = "Example Sequence Alert Rule"
    trigger {
      interval = "1m"
    }
    expressions = {
      "A" = jsonencode({
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
      })
    }
    no_data_state  = "KeepLast"
    exec_err_state = "KeepLast"
  }
}

resource "grafana_apps_rules_rulesequence_v0alpha1" "example" {
  metadata {
    uid        = "example-rule-sequence"
    folder_uid = grafana_folder.rulesequence_folder.uid
  }
  spec {
    # The interval at which every rule in the sequence is evaluated.
    trigger {
      interval = "1m"
    }
    # Recording rules are evaluated first, in the order listed. At least one is required.
    recording_rules = [
      { name = grafana_apps_rules_recordingrule_v0alpha1.example.metadata[0].uid },
    ]
    # Alert rules are evaluated after the recording rules, in the order listed.
    alerting_rules = [
      { name = grafana_apps_rules_alertrule_v0alpha1.example.metadata[0].uid },
    ]
  }
}
