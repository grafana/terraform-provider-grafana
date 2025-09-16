resource "grafana_apps_alertenrichment_alertenrichment_v1beta1" "enrichment" {
  metadata {
    uid = "test_enrichment"
  }

  spec {
    title       = "Comprehensive alert enrichment"
    description = "Demonstrates many enrichment steps and configurations"

    # Target specific alert rules by UIDs
    alert_rule_uids = ["alert-rule-1", "alert-rule-2"]

    # Target specific receiver names
    receivers = ["webhook", "slack-critical"]

    # Label matchers - supports =, !=, =~, !~ operators
    label_matchers = [
      {
        type  = "="
        name  = "severity"
        value = "critical"
      },
      {
        type  = "=~"
        name  = "team"
        value = "alerting|alerting-team"
      }
    ]

    # Annotation matchers
    annotation_matchers = [
      {
        type  = "!="
        name  = "runbook_url"
        value = ""
      }
    ]

    # Adds annotations to alerts with the assign step
    step {
      assign {
        timeout = "30s"
        annotations = {
          "priority"    = "high"
          "runbook_url" = "https://runbooks.grafana.com/alert-handling"
        }
      }
    }

    # Calls external service
    step {
      external {
        url = "https://some-api.grafana.com/alert-enrichment"
      }
    }

    # Data source step with logs query
    step {
      data_source {
        timeout = "30s"

        logs_query {
          data_source_type = "loki"
          data_source_uid  = "loki-uid-123"
          expr             = "{job=\"my-app\"} |= \"error\""
          max_lines        = 5
        }
      }
    }

    # Data source step with raw query
    step {
      data_source {
        timeout = "30s"

        raw_query {
          ref_id = "A"
          request = jsonencode({
            datasource = {
              type = "prometheus"
              uid  = "prometheus-uid-456"
            }
            expr          = "rate(http_requests_total[5m])"
            refId         = "A"
            intervalMs    = 1000
            maxDataPoints = 43200
          })
        }
      }
    }

    # Triggers a new Sift investigation
    step {
      sift {}
    }

    # Generates AI explanation of the alert using Grafana LLM plugin
    step {
      explain {
        annotation = "ai_explanation"
      }
    }


    # Trigger a new Assistant Investigation
    step {
      assistant_investigations {}
    }

    # Conditional step runs different actions based on alert severity
    step {
      conditional {
        # Condition: Check if severity is critical
        if {
          label_matchers = [{
            type  = "="
            name  = "severity"
            value = "critical"
          }]
        }

        # Actions for critical alerts
        then {
          step {
            assign {
              annotations = {
                escalation_level = "immediate"
              }
            }
          }
          step {
            external {
              url = "https://irm.grafana.com/create-incident"
            }
          }
        }

        # Actions for non-critical alerts
        else {
          step {
            assign {
              annotations = {
                escalation_level = "standard"
              }
            }
          }
        }
      }
    }
  }
}
