resource "grafana_folder" "rule_folder" {
    title = "My Alert Rule Folder"
}

resource "grafana_alert_rule" "my_alert_rule" {
    name = "My Alert Rule Group"
    folder_uid = grafana_folder.rule_folder.uid
    interval_seconds = 60
    rules {
        name = "My Alert Rule"
        for = 120
        condition = "TODO"
    }
}
