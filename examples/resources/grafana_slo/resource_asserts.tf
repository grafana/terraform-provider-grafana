resource "grafana_slo" "asserts_example" {
  name        = "Asserts SLO Example"
  description = "SLO managed by Asserts for entity-centric monitoring and RCA"
  query {
    freeform {
      query = "sum(rate(apiserver_request_total{code!=\"500\"}[$__rate_interval])) / sum(rate(apiserver_request_total[$__rate_interval]))"
    }
    type = "freeform"
  }
  objectives {
    value  = 0.995
    window = "30d"
  }
  destination_datasource {
    uid = "grafanacloud-prom"
  }

  # Asserts integration labels
  # The grafana_slo_provenance label triggers Asserts-specific behavior:
  # - Displays "asserts" badge instead of "provisioned"
  # - Shows "Open RCA workbench" button in the SLO UI
  # - Enables correlation with Asserts entity-centric monitoring
  label {
    key   = "grafana_slo_provenance"
    value = "asserts"
  }
  label {
    key   = "service_name"
    value = "my-service"
  }
  label {
    key   = "team_name"
    value = "platform-team"
  }

  # Search expression for Asserts RCA workbench
  # This enables the "Open RCA workbench" button to deep-link with pre-filtered context
  search_expression = "service=my-service"

  alerting {
    fastburn {
      annotation {
        key   = "name"
        value = "SLO Burn Rate Very High"
      }
      annotation {
        key   = "description"
        value = "Error budget is burning too fast"
      }
    }

    slowburn {
      annotation {
        key   = "name"
        value = "SLO Burn Rate High"
      }
      annotation {
        key   = "description"
        value = "Error budget is burning too fast"
      }
    }
  }
}

