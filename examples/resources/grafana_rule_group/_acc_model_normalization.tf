resource "grafana_folder" "rule_folder" {
  title = "Model Normalization Tests Folder"
}

resource "grafana_rule_group" "rg_model_params_defaults" {
  name             = "My Rule Group 1"
  folder_uid       = grafana_folder.rule_folder.uid
  interval_seconds = 240
  org_id           = 1
  rule {
    name           = "My Alert Rule 1"
    for            = "2m"
    condition      = "B"
    no_data_state  = "NoData"
    exec_err_state = "Alerting"
    annotations = {
      "a" = "b"
      "c" = "d"
    }
    labels = {
      "e" = "f"
      "g" = "h"
    }
    is_paused = false
    data {
      ref_id     = "A"
      query_type = ""
      relative_time_range {
        from = 600
        to   = 0
      }
      datasource_uid = "PD8C576611E62080A"
      model = jsonencode({
        hide          = false
        intervalMs    = 1000
        maxDataPoints = 43200
        refId         = "A"
      })
    }
  }
}

resource "grafana_rule_group" "rg_model_params_omitted" {
  name             = "My Rule Group 2"
  folder_uid       = grafana_folder.rule_folder.uid
  interval_seconds = 240
  org_id           = 1
  rule {
    name           = "My Alert Rule 2"
    for            = "2m"
    condition      = "B"
    no_data_state  = "NoData"
    exec_err_state = "Alerting"
    annotations = {
      "a" = "b"
      "c" = "d"
    }
    labels = {
      "e" = "f"
      "g" = "h"
    }
    is_paused = false
    data {
      ref_id     = "A"
      query_type = ""
      relative_time_range {
        from = 600
        to   = 0
      }
      datasource_uid = "PD8C576611E62080A"
      model = jsonencode({
        hide  = false
        refId = "A"
      })
    }
  }
}

resource "grafana_rule_group" "rg_model_params_non_default" {
  name             = "My Rule Group 3"
  folder_uid       = grafana_folder.rule_folder.uid
  interval_seconds = 240
  org_id           = 1
  rule {
    name           = "My Alert Rule 3"
    for            = "2m"
    condition      = "B"
    no_data_state  = "NoData"
    exec_err_state = "Alerting"
    annotations = {
      "a" = "b"
      "c" = "d"
    }
    labels = {
      "e" = "f"
      "g" = "h"
    }
    is_paused = false
    data {
      ref_id     = "A"
      query_type = ""
      relative_time_range {
        from = 600
        to   = 0
      }
      datasource_uid = "PD8C576611E62080A"
      model = jsonencode({
        hide          = false
        intervalMs    = 1001
        maxDataPoints = 43201
        refId         = "A"
      })
    }
  }
}
