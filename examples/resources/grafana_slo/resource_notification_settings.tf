resource "grafana_contact_point" "slo_oncall" {
  name = "SLO On-call"

  email {
    addresses = ["oncall@example.com"]
  }
}

resource "grafana_slo" "notification_settings" {
  name        = "Terraform Testing - Notification settings"
  description = "Routes each burn rate's alerts to a specific contact point"
  query {
    ratio {
      success_metric  = "kubelet_http_requests_total{status!~\"5..\"}"
      total_metric    = "kubelet_http_requests_total"
      group_by_labels = ["job", "instance"]
    }
    type = "ratio"
  }
  objectives {
    value  = 0.995
    window = "30d"
  }
  destination_datasource {
    uid = "grafanacloud-prom"
  }
  alerting {
    fastburn {
      annotation {
        key   = "name"
        value = "SLO Burn Rate Very High"
      }
      # Send these alerts to a contact point...
      notification_settings {
        contact_point = grafana_contact_point.slo_oncall.name
      }
    }

    slowburn {
      annotation {
        key   = "name"
        value = "SLO Burn Rate High"
      }
      # ...or through a named notification policy. Set one of contact_point or policy, not both.
      # Omit the block to use the default notification policy.
      notification_settings {
        policy = "team-a"
      }
    }
  }
}
