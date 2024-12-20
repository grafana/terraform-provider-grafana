resource "grafana_organization" "test" {
  name = "alerting-org"
}

resource "grafana_contact_point" "test" {
  org_id = grafana_organization.test.id
  name   = "my-contact-point"
  email {
    addresses = ["hello@example.com"]
  }
}

resource "grafana_notification_policy" "test" {
  org_id        = grafana_organization.test.id
  contact_point = grafana_contact_point.test.name
  group_by      = ["..."]
}

resource "grafana_mute_timing" "my_mute_timing" {
  org_id = grafana_organization.test.id
  name   = "My Mute Timing"

  intervals {
    times {
      start = "04:56"
      end   = "14:17"
    }
    weekdays      = ["monday", "tuesday:thursday"]
    days_of_month = ["1:7", "-1"]
    months        = ["1:3", "december"]
    years         = ["2030", "2025:2026"]
    location      = "America/New_York"
  }
}

resource "grafana_message_template" "my_template" {
  org_id   = grafana_organization.test.id
  name     = "My Reusable Template"
  template = "{{ define \"My Reusable Template\" }}\n template content\n{{ end }}"
}

resource "grafana_folder" "rule_folder" {
  org_id = grafana_organization.test.id
  title  = "My Alert Rule Folder"
  uid    = "alert-rule-folder"
}

resource "grafana_rule_group" "my_alert_rule" {
  org_id           = grafana_organization.test.id
  name             = "My Rule Group"
  folder_uid       = grafana_folder.rule_folder.uid
  interval_seconds = 240
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
}
