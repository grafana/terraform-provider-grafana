terraform {
  required_providers {
    grafana = {
      source  = "grafana.com/grafana/grafana"
      version = "0.1"
    }
  }
}

provider "grafana" {
  url    = "http://localhost:3000"
  auth   = ""
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

resource "grafana_panel" "panel1" {
  temp_json = <<EOT
    {
      "datasource": {
        "type": "testdata",
        "uid": "PD8C576611E62080A"
      },
      "fieldConfig": {
        "defaults": {
          "color": {
            "mode": "palette-classic"
          },
          "custom": {
            "axisCenteredZero": false,
            "axisColorMode": "text",
            "axisLabel": "",
            "axisPlacement": "auto",
            "barAlignment": 0,
            "drawStyle": "line",
            "fillOpacity": 0,
            "gradientMode": "none",
            "hideFrom": {
              "legend": false,
              "tooltip": false,
              "viz": false
            },
            "lineInterpolation": "linear",
            "lineWidth": 1,
            "pointSize": 5,
            "scaleDistribution": {
              "type": "linear"
            },
            "showPoints": "auto",
            "spanNulls": false,
            "stacking": {
              "group": "A",
              "mode": "none"
            },
            "thresholdsStyle": {
              "mode": "off"
            }
          },
          "mappings": [],
          "thresholds": {
            "mode": "absolute",
            "steps": [
              {
                "color": "green",
                "value": null
              },
              {
                "color": "red",
                "value": 80
              }
            ]
          }
        },
        "overrides": []
      },
      "gridPos": {
        "h": 8,
        "w": 12,
        "x": 0,
        "y": 0
      },
      "id": 2,
      "options": {
        "legend": {
          "calcs": [],
          "displayMode": "list",
          "placement": "bottom",
          "showLegend": true
        },
        "tooltip": {
          "mode": "single",
          "sort": "none"
        }
      },
      "title": "Panel Created With Terraform",
      "type": "timeseries"
    }
    EOT
}

resource "grafana_dashboard" "dashboard_terraform" {
  panels {
    config_json = grafana_panel.panel1.config_json
  }
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
  "title": "Dashboard Two",
  "uid": "d340ebbb-7234-44d8-b5ea-c3c78e3b4baa",
  "weekStart": ""
}
EOT
}
