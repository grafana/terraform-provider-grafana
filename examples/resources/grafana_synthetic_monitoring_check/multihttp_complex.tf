data "grafana_synthetic_monitoring_probes" "main" {}

resource "grafana_synthetic_monitoring_check" "multihttp" {
  job     = "multihttp complex"
  target  = "https://www.an-auth-endpoint.com"
  enabled = false
  probes = [
    data.grafana_synthetic_monitoring_probes.main.probes.Amsterdam,
  ]
  labels = {
    foo = "bar"
  }
  settings {
    multihttp {
      entries {
        request {
          method = "POST"
          url    = "https://www.an-auth-endpoint.com"
          query_fields {
            name  = "username"
            value = "steve"
          }
          query_fields {
            name  = "password"
            value = "top_secret"
          }
          body {
            content_type = "application/json"
          }
        }
        checks {
          type      = 0
          subject   = 2
          condition = 2
          value     = "200"
        }
        variables {
          type       = 0
          name       = "accessToken"
          expression = "data.accessToken"
        }
      }
      entries {
        request {
          method = "GET"
          url    = "https://www.an-endpoint-that-requires-auth.com"
          headers {
            name  = "Authorization"
            value = "Bearer $${accessToken}"
          }
        }
        checks {
          type       = 1
          condition  = 6
          expression = "result"
          value      = "expected"
        }
      }
    }
  }
}
