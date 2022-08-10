resource "grafana_folder" "rule_folder" {
    title = "My Alert Rule Folder"
}

resource "grafana_alert_rule" "my_alert_rule" {
    name = "My Rule Group"
    folder_uid = grafana_folder.rule_folder.uid
    interval_seconds = 60
    org_id = 1
    rules {
        name = "My Alert Rule 1"
        for = 120
        condition = "B"
        no_data_state = "NoData"
        exec_err_state = "Alerting"
        annotations = {
            "a" = "b"
            "c" = "d"
        }
        labels = {
            "e" = "f"
            "g" = "h"
        }
        data {
            ref_id = "A"
            query_type = ""
            relative_time_range {
                from = 600
                to = 0
            }
            datasource_uid = "PD8C576611E62080A"
            model = jsonencode({
                hide = false
                intervalMs = 1000
                maxDataPoints = 43200
                refId = "A"
            })
        }
        data {
            ref_id = "B"
            query_type = ""
            relative_time_range {
                from = 0
                to = 0
            }
            datasource_uid = "-100"
            model = <<EOT
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
}
