data "grafana_synthetic_monitoring_probes" "main" {}

resource "grafana_synthetic_monitoring_check" "multihttp" {
  job     = "multihttp complex"
  target  = "https://www.an-auth-endpoint.com"
  enabled = false
  probes = [
    data.grafana_synthetic_monitoring_probes.main.probes.Frankfurt,
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
        assertions {
          type      = "TEXT"
          subject   = "HTTP_STATUS_CODE"
          condition = "EQUALS"
          value     = "200"
        }
        variables {
          type       = "JSON_PATH"
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
        assertions {
          type      = "TEXT"
          subject   = "RESPONSE_BODY"
          condition = "CONTAINS"
          value     = "foobar"
        }
        assertions {
          type      = "TEXT"
          subject   = "RESPONSE_BODY"
          condition = "NOT_CONTAINS"
          value     = "xyyz"
        }
        assertions {
          type       = "JSON_PATH_VALUE"
          condition  = "EQUALS"
          expression = "$.slideshow.author"
          value      = "Yours Truly"
        }
        assertions {
          type       = "JSON_PATH_VALUE"
          condition  = "STARTS_WITH"
          expression = "$.slideshow.date"
          value      = "date of "
        }
        assertions {
          type       = "JSON_PATH_ASSERTION"
          expression = "$.slideshow.slides"
        }
      }
    }
  }
}
