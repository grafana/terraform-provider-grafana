resource "grafana_apps_notifications_receiver_v1beta1" "operations" {
  metadata {
    uid = "operations"
  }

  spec {
    title = "Operations"

    # Integrations are grouped by type. Each one needs a uid that is unique across all receivers.
    slack {
      uid   = "operations-slack-primary"
      url   = "https://example.com/slack-primary-webhook"
      title = "Primary notifications"
    }

    slack {
      uid                     = "operations-slack-secondary"
      url                     = "https://example.com/slack-secondary-webhook"
      disable_resolve_message = true
    }

    email {
      uid       = "operations-email"
      addresses = ["operations@example.com"]
    }

    webhook {
      uid         = "operations-webhook"
      url         = "https://example.com/hook"
      http_method = "POST"
      max_alerts  = 10

      http_config {
        oauth2 {
          client_id     = "grafana"
          client_secret = "replace-me"
          token_url     = "https://example.com/oauth/token"
        }
      }
    }
  }
}
