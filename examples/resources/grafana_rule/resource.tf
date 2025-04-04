resource "grafana_folder" "rule_folder" {
  title = "My Alert Rule Folder"
}

resource "grafana_rule" "test_rule" {
  disable_provenance = true
  name               = "My Alert Rule"
  folder_uid         = grafana_folder.rule_folder.uid
  rule_group         = "My Rule Group"
  for                = "2m"
  condition          = "B"
  no_data_state      = "NoData"
  exec_err_state     = "Alerting"
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
  data {
    ref_id     = "B"
    query_type = ""
    relative_time_range {
      from = 0
      to   = 0
    }
    datasource_uid = "-100"
    model          = <<EOT
{
    "conditions": [
        {
        "evaluator": {
            "params": [
            3
            ],
            "type": "gt"
        },
        "operator": {
            "type": "and"
        },
        "query": {
            "params": [
            "A"
            ]
        },
        "reducer": {
            "params": [],
            "type": "last"
        },
        "type": "query"
        }
    ],
    "datasource": {
        "type": "__expr__",
        "uid": "-100"
    },
    "hide": false,
    "intervalMs": 1000,
    "maxDataPoints": 43200,
    "refId": "B",
    "type": "classic_conditions"
}
EOT
  }
}
