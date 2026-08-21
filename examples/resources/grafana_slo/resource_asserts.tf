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

  # Knowledge Graph search expression for the Asserts RCA workbench.
  # This is what enables the "Open RCA workbench" link to deep-link with
  # pre-filtered context -- it is driven by the expression, not by the
  # provenance label above.
  search_expression = "my-service connected services"

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

