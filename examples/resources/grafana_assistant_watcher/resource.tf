# Calibrate the watcher once in the Assistant UI, then copy the resulting
# checks and calibration baseline into this resource to manage it as code.
resource "grafana_assistant_watcher" "checkout_service" {
  name        = "Checkout Service"
  description = "Availability, errors and latency for the checkout flow."

  prompt = <<-EOT
    Monitor the checkoutservice in the ecommerce-prod namespace: availability,
    gRPC errors, latency, dependency failures, and error logs. Errors should be
    rare; do not treat elevated error signals as expected noise without evidence
    they are chronic and inconsequential.
  EOT

  datasource_uids          = ["prometheus-uid", "loki-uid"]
  sensitivity              = "balanced"
  trigger_interval_seconds = 900
  started                  = true

  calibration_context = <<-EOT
    checkoutservice orchestrates the ecommerce-prod checkout flow across three
    regional replicas. Treat the regions as one service, but call out a single
    unhealthy region while the others are fine.

    KNOWN BENIGN LOG NOISE: the "ORDER_API_URL is not configured" warning fires
    continuously with no measured impact. Do not flag it.
  EOT

  query {
    type           = "promql"
    datasource_uid = "prometheus-uid"
    expr           = "clamp_max(sum(rate(rpc_server_duration_milliseconds_count{service_name=\"checkoutservice\", rpc_grpc_status_code!=\"0\"}[15m])) / clamp_min(sum(rate(rpc_server_duration_milliseconds_count{service_name=\"checkoutservice\"}[15m])), 0.001), 1)"
    comment        = "gRPC error ratio for checkoutservice (0-1 fraction)"
    role           = "fast_incident"
    good_when      = "low"

    thresholds {
      comparator = "gt"
      warning    = 0.05
      critical   = 0.15
      source     = "alert CheckoutServiceErrorRate (5% warn); critical derived 3x"
    }
  }

  # An alerts check reads Grafana-managed Alertmanager, so it needs no
  # datasource UID. A firing alert is itself the signal, so it needs no
  # numeric thresholds either.
  query {
    type    = "alerts"
    expr    = "alertname=\"CheckoutServiceP95Latency\""
    comment = "Firing status of the checkout p95 latency alert"
    role    = "current_health"
  }

  query {
    type           = "logql"
    datasource_uid = "loki-uid"
    expr           = "sum(count_over_time({service_name=\"checkoutservice\"} |= \"Error pinging orders database\" | detected_level=\"error\" [15m]))"
    comment        = "Count of orders-db connection error log lines"
    role           = "fast_incident"
    good_when      = "low"

    thresholds {
      comparator = "gt"
      warning    = 6
      critical   = 15
      source     = "derived, scaled from alert OrdersDatabaseConnectionErrors (>2/5m)"
    }
  }

  actions {
    slack {
      enabled = true

      target {
        type       = "channel"
        channel_id = "C0123456789"
      }
    }

    investigation {
      enabled    = true
      team_names = ["commerce"]
    }
  }
}
