resource "grafana_apps_notifications_routingtree_v1beta1" "example" {
  metadata {
    # Each routing tree has a user-chosen name. Multiple named routing trees
    # are supported (requires the alertingMultiplePolicies feature toggle).
    uid = "my-routing-tree"
  }

  spec {
    # Set to true to allow editing this tree outside Terraform (e.g. the Grafana UI).
    disable_provenance = false

    # Defaults applied to alerts that do not match any specific route.
    defaults {
      receiver        = "empty"
      group_by        = ["grafana_folder", "alertname"]
      group_wait      = "30s"
      group_interval  = "5m"
      repeat_interval = "4h"
    }

    # A specific route for critical alerts.
    routes {
      receiver = "empty"
      continue = false

      matchers = [
        {
          type  = "="
          label = "severity"
          value = "critical"
        }
      ]

      mute_time_intervals = []
      group_by            = ["alertname"]

      # Routes can be nested.
      routes {
        receiver = "empty"
        continue = true

        matchers = [
          {
            type  = "=~"
            label = "team"
            value = "backend|platform"
          }
        ]
      }
    }
  }
}
