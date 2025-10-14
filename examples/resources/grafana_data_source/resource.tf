resource "grafana_data_source" "arbitrary-data" {
  type = "stackdriver"
  name = "sd-arbitrary-data"

  json_data_encoded = jsonencode({
    "tokenUri"           = "https://oauth2.googleapis.com/token"
    "authenticationType" = "jwt"
    "defaultProject"     = "default-project"
    "clientEmail"        = "client-email@default-project.iam.gserviceaccount.com"
  })

  secure_json_data_encoded = jsonencode({
    "privateKey" = "-----BEGIN PRIVATE KEY-----\nprivate-key\n-----END PRIVATE KEY-----\n"
  })
}

resource "grafana_data_source" "influxdb" {
  type                = "influxdb"
  name                = "myapp-metrics"
  url                 = "http://influxdb.example.net:8086/"
  basic_auth_enabled  = true
  basic_auth_username = "username"
  database_name       = "dbname" // Example: influxdb_database.metrics.name

  json_data_encoded = jsonencode({
    authType          = "default"
    basicAuthPassword = "mypassword"
  })
}

resource "grafana_data_source" "cloudwatch" {
  type = "cloudwatch"
  name = "cw-example"

  json_data_encoded = jsonencode({
    defaultRegion = "us-east-1"
    authType      = "keys"
  })

  secure_json_data_encoded = jsonencode({
    accessKey = "123"
    secretKey = "456"
  })
}

resource "grafana_data_source" "cloudwatch_assumeARN" {
  type = "cloudwatch"
  name = "cw-assumeARN-example"

  # Requires `assume_role_enabled` feature flag to be enabled
  # OSS: use authType = "default" on OSS
  # Cloud: use authType = "grafana_assume_role" which is in private preview on Cloud:
  # https://grafana.com/docs/grafana/latest/datasources/aws-cloudwatch/aws-authentication/#use-grafana-assume-role
  json_data_encoded = jsonencode({
    defaultRegion = "us-east-1"
    authType      = "grafana_assume_role"
    assumeRoleArn = "arn:aws:iam::123456789012:root"
  })
}

resource "grafana_data_source" "prometheus" {
  type                = "prometheus"
  name                = "mimir"
  url                 = "https://my-instances.com"
  basic_auth_enabled  = true
  basic_auth_username = "username"

  json_data_encoded = jsonencode({
    httpMethod        = "POST"
    prometheusType    = "Mimir"
    prometheusVersion = "2.4.0"
  })

  secure_json_data_encoded = jsonencode({
    basicAuthPassword = "password"
  })
}

