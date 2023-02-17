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

resource "grafana_data_source" "terraform_postgres" {
  type                = "postgres"
  name                = "terraform-postgres"
  uid                 = "terraform-postgres-uid"
  url                 = "localhost:5432"
  username = "grafana"
  database_name = "grafana"
  json_data_encoded   = jsonencode({
    database = "grafana"
    sslmode        = "disable"
  })
  secure_json_data_encoded = jsonencode({
    password = "password"
  })
}

data "grafana_table_panel" "terraform_table" {
  title = "Table with Terraform"
  datasource_type = grafana_data_source.terraform_postgres.type
  datasource_uid = grafana_data_source.terraform_postgres.uid
  target_json =<<EOT
[
    {
      "editorMode": "builder",
      "format": "table",
      "rawSql": "SELECT value FROM grafana_metric LIMIT 50 ",
      "refId": "A",
      "sql": {
        "columns": [
          {
            "parameters": [
              {
                "name": "value",
                "type": "functionParameter"
              }
            ],
            "type": "function"
          }
        ],
        "groupBy": [
          {
            "property": {
              "type": "string"
            },
            "type": "groupBy"
          }
        ],
        "limit": 50
      },
      "table": "grafana_metric"
    }
  ]
EOT
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
  panels {
    config_json = data.grafana_table_panel.terraform_table.config_json
    grid_pos {
      h = 8
      w = 8
      x = 0
      y = 9
    }
  }
  depends_on = [grafana_data_source.terraform_postgres]
}
