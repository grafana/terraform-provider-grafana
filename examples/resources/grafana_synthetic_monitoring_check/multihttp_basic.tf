data "grafana_synthetic_monitoring_probes" "main" {}

resource "grafana_synthetic_monitoring_check" "multihttp" {
  job = "MultiHTTP defaults"
  target = "https://www.grafana-dev.com"
  enabled = false
  probes = [
    data.grafana_synthetic_monitoring_probes.main.probes.Atlanta,
  ]        
  labels = {
    foo = "bar"
  }
  settings {
    multihttp {
      entries {
        request {
          method = "GET"
          url = "https://www.grafana-dev.com"
        }
      }
    }
  }
}
      #       "entries": [
      #         {
      #           "request": {
      #             "method": "GET",
      #             "url": "https://www.grafana-dev.com"
      #           },
      #           "checks": [
      #             {
      #               "type": 0,
      #               "subject": 2,
      #               "condition": 2,
      #               "value": "200"
      #             },
      #             {
      #               "type": 0,
      #               "subject": 2,
      #               "condition": 2,
      #               "value": "300"
      #             }
      #           ],
      #           "variables": [
      #             {
      #               "type": 0,
      #               "name": "accessToken",
      #               "expression": "accessToken"
      #             }
      #           ]
      #         }
      #       ]
      #     }
      #   }
      # },