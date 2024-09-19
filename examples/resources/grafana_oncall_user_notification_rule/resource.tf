data "grafana_oncall_user" "my_user" {
  provider = grafana.oncall
  username = "my_username"
}

resource "grafana_oncall_user_notification_rule" "my_user_step_1" {
  provider = grafana.oncall
  user_id  = data.grafana_oncall_user.my_user.id
  position = 0
  type     = "notify_by_mobile_app"
}

resource "grafana_oncall_user_notification_rule" "my_user_step_2" {
  provider = grafana.oncall
  user_id  = data.grafana_oncall_user.my_user.id
  position = 1
  duration = 600 # 10 minutes
  type     = "wait"
}

resource "grafana_oncall_user_notification_rule" "my_user_step_3" {
  provider = grafana.oncall
  user_id  = data.grafana_oncall_user.my_user.id
  position = 2
  type     = "notify_by_phone_call"
}

resource "grafana_oncall_user_notification_rule" "my_user_step_4" {
  provider = grafana.oncall
  user_id  = data.grafana_oncall_user.my_user.id
  position = 3
  duration = 300 # 5 minutes
  type     = "wait"
}

resource "grafana_oncall_user_notification_rule" "my_user_step_5" {
  provider = grafana.oncall
  user_id  = data.grafana_oncall_user.my_user.id
  position = 4
  type     = "notify_by_slack"
}

resource "grafana_oncall_user_notification_rule" "my_user_important_step_1" {
  provider  = grafana.oncall
  user_id   = data.grafana_oncall_user.my_user.id
  important = true
  position  = 0
  type      = "notify_by_mobile_app_critical"
}

resource "grafana_oncall_user_notification_rule" "my_user_important_step_2" {
  provider  = grafana.oncall
  user_id   = data.grafana_oncall_user.my_user.id
  important = true
  position  = 1
  duration  = 300 # 5 minutes
  type      = "wait"
}

resource "grafana_oncall_user_notification_rule" "my_user_important_step_3" {
  provider  = grafana.oncall
  user_id   = data.grafana_oncall_user.my_user.id
  important = true
  position  = 2
  type      = "notify_by_mobile_app_critical"
}
