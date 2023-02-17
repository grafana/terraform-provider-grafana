terraform {
  required_providers {
    grafana = {
      source  = "grafana.com/test/grafana"
      version = "0.1"
    }
  }
}

provider "grafana" {
  url    = "http://localhost:3000"
  auth   = "admin:admin"
  org_id = 1
}

#resource "grafana_data_source" "prometheus" {
#  type                = "prometheus"
#  name                = "prometheus-ds-test"
#  uid                 = "prometheus-ds-test-uid"
#  url                 = "https://my-instance.com"
#  basic_auth_enabled  = true
#  basic_auth_username = "username"
#  json_data_encoded   = jsonencode({
#    httpMethod        = "POST"
#    prometheusType    = "Mimir"
#    prometheusVersion = "2.4.0"
#  })
#  secure_json_data_encoded = jsonencode({
#    basicAuthPassword = "password"
#  })
#}

data "grafana_text_panel" "welcome_panel" {
  content = <<EOT
    <h2 style=\"text-align: center;\">Welcome to Grafana Kinds</h2>
    <p style=\"text-align: center;\">Welcome to Grafana Kinds. Kinds are a representation for objects that includes a schema and some other properties, like the maturity level.</p>
    <p style=\"text-align: center;\">Maturity is an important part of Kinds, because it relates to how confident we are that the schema can be used in Grafana.</p>
    <p style=\"text-align: center;\">For more information on maturity levels, you can read <a href="https://docs.google.com/document/d/1gPzSCQn-0Qg86PYpPcR-l_kTQ4Ba1wgnNLZFKPSe8yQ/edit#heading=h.7d65grgc1wl6">this document</a>.</p>
  EOT
  transparent = true
  mode = "html"
}

resource "grafana_dashboard" "dashboard_terraform" {
  config_json = <<EOT
{
  "annotations": {
    "list": [
      {
        "builtIn": 1,
        "datasource": {
          "type": "grafana",
          "uid": "-- Grafana --"
        },
        "enable": true,
        "hide": true,
        "iconColor": "rgba(0, 211, 255, 1)",
        "name": "Annotations & Alerts",
        "target": {
          "limit": 100,
          "matchAny": false,
          "tags": [],
          "type": "dashboard"
        },
        "type": "dashboard"
      }
    ]
  },
  "editable": true,
  "fiscalYearStartMonth": 0,
  "graphTooltip": 0,
  "links": [],
  "liveNow": false,
  "panels": [],
  "revision": 1,
  "schemaVersion": 37,
  "style": "dark",
  "tags": [],
  "templating": {
    "list": []
  },
  "time": {
    "from": "now-6h",
    "to": "now"
  },
  "timepicker": {},
  "timezone": "",
  "title": "Grafana Kinds",
  "weekStart": ""
}
EOT

  panels {
    config_json = data.grafana_text_panel.welcome_panel.config_json
    grid_pos {
      h = 8
      w = 8
      x = 5
      y = 0
    }
  }
}
