resource "grafana_data_source" "influxdb" {
  type          = "influxdb"
  name          = "myapp-metrics"
  url           = "http://influxdb.example.net:8086/"
  username      = "myapp"
  password      = "foobarbaz"
  database_name = influxdb_database.metrics.name
}

resource "grafana_data_source" "cloudwatch" {
  type = "cloudwatch"
  name = "cw-example"

  json_data {
    default_region = "us-east-1"
    auth_type      = "keys"
  }

  secure_json_data {
    access_key = "123"
    secret_key = "456"
  }
}

resource "grafana_data_source" "stackdriver" {
  type = "stackdriver"
  name = "sd-example"

  json_data {
    token_uri           = "https://oauth2.googleapis.com/token"
    authentication_type = "jwt"
    default_project     = "default-project"
    client_email        = "client-email@default-project.iam.gserviceaccount.com"
  }

  secure_json_data {
    private_key = "-----BEGIN PRIVATE KEY-----\nprivate-key\n-----END PRIVATE KEY-----\n"
  }
}
