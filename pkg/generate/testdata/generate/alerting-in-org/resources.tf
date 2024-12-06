# __generated__ by Terraform
# Please review these resources and move them into your main configuration files.

# __generated__ by Terraform from "1:email receiver"
resource "grafana_contact_point" "_1_email_receiver" {
  disable_provenance = true
  name               = "email receiver"
  email {
    addresses               = ["<example@email.com>"]
    disable_resolve_message = false
    single_email            = false
  }
}

# __generated__ by Terraform from "2:email receiver"
resource "grafana_contact_point" "_2_email_receiver" {
  disable_provenance = true
  name               = "email receiver"
  org_id             = grafana_organization.alerting-org.id
  email {
    addresses               = ["<example@email.com>"]
    disable_resolve_message = false
    single_email            = false
  }
}

# __generated__ by Terraform from "2:my-contact-point"
resource "grafana_contact_point" "_2_my-contact-point" {
  disable_provenance = false
  name               = "my-contact-point"
  org_id             = grafana_organization.alerting-org.id
  email {
    addresses               = ["hello@example.com"]
    disable_resolve_message = false
    single_email            = false
  }
}

# __generated__ by Terraform from "2:alert-rule-folder"
resource "grafana_folder" "_2_alert-rule-folder" {
  org_id = grafana_organization.alerting-org.id
  title  = "My Alert Rule Folder"
  uid    = "alert-rule-folder"
}

# __generated__ by Terraform from "2:My Notification Template Group"
resource "grafana_message_template" "_2_My_Notification_Template_Group" {
  name     = "My Notification Template Group"
  org_id   = grafana_organization.alerting-org.id
  template = "{{define \"custom.message\" }}\n template content\n{{ end }}"
}

# __generated__ by Terraform from "2:My Mute Timing"
resource "grafana_mute_timing" "_2_My_Mute_Timing" {
  name   = "My Mute Timing"
  org_id = grafana_organization.alerting-org.id
  intervals {
    days_of_month = ["1:7", "-1"]
    location      = "America/New_York"
    months        = ["1:3", "12"]
    weekdays      = ["monday", "tuesday:thursday"]
    years         = ["2030", "2025:2026"]
    times {
      end   = "14:17"
      start = "04:56"
    }
  }
}

# __generated__ by Terraform from "1:policy"
resource "grafana_notification_policy" "_1_policy" {
  contact_point      = "grafana-default-email"
  disable_provenance = true
  group_by           = ["grafana_folder", "alertname"]
}

# __generated__ by Terraform from "2:policy"
resource "grafana_notification_policy" "_2_policy" {
  contact_point      = grafana_contact_point._2_my-contact-point.name
  disable_provenance = false
  group_by           = ["..."]
  org_id             = grafana_organization.alerting-org.id
}

# __generated__ by Terraform from "2"
resource "grafana_organization" "alerting-org" {
  admins = ["admin@localhost"]
  name   = "alerting-org"
}

# __generated__ by Terraform from "2:alert-rule-folder:My Rule Group"
resource "grafana_rule_group" "_2_alert-rule-folder_My_Rule_Group" {
  disable_provenance = false
  folder_uid         = grafana_folder._2_alert-rule-folder.uid
  interval_seconds   = 240
  name               = "My Rule Group"
  org_id             = grafana_organization.alerting-org.id
  rule {
    annotations = {
      a = "b"
      c = "d"
    }
    condition      = "B"
    exec_err_state = "Alerting"
    for            = "2m0s"
    is_paused      = false
    labels = {
      e = "f"
      g = "h"
    }
    name          = "My Alert Rule 1"
    no_data_state = "NoData"
    data {
      datasource_uid = "PD8C576611E62080A"
      model = jsonencode({
        hide  = false
        refId = "A"
      })
      ref_id = "A"
      relative_time_range {
        from = 600
        to   = 0
      }
    }
    data {
      datasource_uid = jsonencode(-100)
      model = jsonencode({
        conditions = [{
          evaluator = {
            params = [3]
            type   = "gt"
          }
          operator = {
            type = "and"
          }
          query = {
            params = ["A"]
          }
          reducer = {
            params = []
            type   = "last"
          }
          type = "query"
        }]
        datasource = {
          type = "__expr__"
          uid  = "-100"
        }
        hide  = false
        refId = "B"
        type  = "classic_conditions"
      })
      ref_id = "B"
      relative_time_range {
        from = 0
        to   = 0
      }
    }
  }
}
