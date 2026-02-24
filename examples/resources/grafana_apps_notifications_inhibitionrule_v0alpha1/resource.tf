resource "grafana_apps_notifications_inhibitionrule_v0alpha1" "example" {
  metadata {
    uid = "example-inhibition-rule"
  }

  spec {
    # Matchers for the source alert (the one that suppresses others)
    source_matchers = [
      {
        type  = "="
        label = "alertname"
        value = "TargetDown"
      }
    ]

    # Matchers for the target alert (the one being suppressed)
    target_matchers = [
      {
        type  = "="
        label = "severity"
        value = "warning"
      }
    ]

    # Labels that must have equal values in source and target alerts
    equal = ["namespace", "pod"]
  }
}
