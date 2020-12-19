---
layout: "grafana"
page_title: "Grafana: grafana_data_source"
sidebar_current: "docs-grafana-resource-data-source"
description: |-
  The grafana_data_source resource allows a Grafana data source to be created.
---

# grafana\_data\_source

The data source resource allows a data source to be created on a Grafana server.

## Example Usage

The required arguments for this resource vary depending on the type of data
source selected (via the `type` argument). The following examples are for
InfluxDB, CloudWatch, and Google Stackdriver. See [Grafana's Data Sources Guides][datasources] for more details on
the supported data source types and the arguments they use.

[datasources]: https://grafana.com/docs/grafana/latest/datasources/#data-sources

For an InfluxDB datasource:

```hcl
resource "grafana_data_source" "metrics" {
  type          = "influxdb"
  name          = "myapp-metrics"
  url           = "http://influxdb.example.net:8086/"
  username      = "myapp"
  password      = "foobarbaz"
  database_name = "${influxdb_database.metrics.name}"
}
```

For a CloudWatch datasource:

```hcl
resource "grafana_data_source" "test_cloudwatch" {
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
```

For a Stackdriver datasource:

```hcl
resource "grafana_data_source" "test_stackdriver" {
  type = "stackdriver"
  name = "sd-example"

  secure_json_data {
    private_key = "-----BEGIN PRIVATE KEY-----\nprivate-key\n-----END PRIVATE KEY-----\n"
  }
}
```

## Argument Reference

The following arguments are supported:

* `type` - (Required) The data source type. Must be one of the data source
  keywords supported by the Grafana server.

* `name` - (Required) A unique name for the data source within the Grafana
  server.

* `url` - (Optional) The URL for the data source. The type of URL required
  varies depending on the chosen data source type.

* `is_default` - (Optional) If true, the data source will be the default
  source used by the Grafana server. Only one data source on a server can be
  the default.

* `basic_auth_enabled` - (Optional) - If true, HTTP basic authentication will
  be used to make requests.

* `basic_auth_username` - (Required if `basic_auth_enabled` is true) The
  username to use for basic auth.

* `basic_auth_password` - (Required if `basic_auth_enabled` is true) The
  password to use for basic auth.

* `username` - (Required by some data source types) The username to use to
  authenticate to the data source.

* `password` - (Required by some data source types) The password to use to
  authenticate to the data source.

* `json_data` - (Required by some data source types) The default region
  and authentication type to access the data source. `json_data` is documented
  in more detail below.

* `secure_json_data` - (Required by some data source types) The access and
  secret keys required to access the data source. `secure_json_data` is
  documented in more detail below.

* `database_name` - (Required by some data source types) The name of the
  database to use on the selected data source server.

* `access_mode` - (Optional) The method by which the browser-based Grafana
  application will access the data source. The default is `proxy`, which means
  that the application will make requests via a proxy endpoint on the Grafana
  server. Proxy is displayed in the Grafana admin as Server. Another possible value is 
  `direct` which is displayed in the Grafana admin as Browser.

JSON Data (`json_data`) supports the following:

All fields are optional, though some data sources may need a subset of these
fields to operate properly.

* `assume_role_arn` - (CloudWatch) The ARN of the role to be assumed by Grafana
  when using the CloudWatch data source.

* `auth_type` - (CloudWatch) The authentication type type used to access the
  data source.

* `conn_max_lifetime` - (MySQL, PostgreSQL, and MSSQL) Maximum amount of time in
  seconds a connection may be reused (Grafana v5.4+).

* `custom_metrics_namespaces` - (CloudWatch)
  A comma-separated list of custom namespaces to be queried by the CloudWatch
  data source.

* `default_region` - (CloudWatch) The default region for the data source.

* `encrypt` - (MSSQL) Connection SSL encryption handling. 'disable', 'false' or
  'true'

* `es_version` - (Elasticsearch) Elasticsearch version as a number (2/5/56/60/70).

* `graphite_version` - (Graphite) Graphite version

* `http_method` - (Prometheus) HTTP method to use for making requests.

* `interval` - (Elasticsearch) Index date time format. nil(No Pattern), 'Hourly',
  'Daily', 'Weekly', 'Monthly' or 'Yearly'.

* `log_level_field` - (Elasticsearch) Which field should be used to indicate the
  priority of the log message.

* `log_message_field` - (Elasticsearch) Which field should be used as the log
  message.

* `max_idle_conns` - (MySQL, PostgreSQL and MSSQL) Maximum number of connections
  in the idle connection pool (Grafana v5.4+).

* `max_open_conns` - (MySQL, PostgreSQL and MSSQL) Maximum number of open
  connections to the database (Grafana v5.4+).

* `postgres_version` - (PostgreSQL) Postgres version as a number
  (903/904/905/906/1000) meaning v9.3, v9.4, …, v10.

* `query_timeout` - (Prometheus) Timeout for queries made to the Prometheus
  data source in seconds.

* `ssl_mode` - (PostgreSQL) SSLmode. 'disable', 'require', 'verify-ca' or
  'verify-full'.

* `timescaledb` - (PostgreSQL) Enable usage of TimescaleDB extension.

* `time_field` - (Elasticsearch) Which field that should be used as timestamp.

* `time_interval` - (Prometheus, Elasticsearch, InfluxDB, MySQL, PostgreSQL, and
  MSSQL) Lowest interval/step value that should be used for this data source.

* `tls_auth` - (All) Enable TLS authentication using client cert configured in
  secure json data.

* `tls_auth_with_ca_cert` - (All) Enable TLS authentication using CA cert.

* `tls_skip_verify` - (All) Controls whether a client verifies the server’s
  certificate chain and host name.

* `tsdb_resolution` - (OpenTSDB) Resolution.

* `tsdb_version` - (OpenTSDB) Version.

Secure JSON Data (`secure_json_data`) supports the following:

All fields are optional, though some data sources may need a subset of these
fields to operate properly.

* `access_key` - (CloudWatch) The access key to use to access the data source.

* `basic_auth_password` - (All) Password to use for basic authentication.

* `password` - (All) Password to use for authentication.

* `private_key` - (Stackdriver) The [service account key](https://cloud.google.com/iam/docs/creating-managing-service-account-keys) `private_key` to use to access the data source.

* `tls_ca_cert` - (All) CA cert for out going requests.

* `tls_client_cert` - (All) TLS Client cert for outgoing requests.

* `tls_client_key` - (All) TLS Client key for outgoing requests.

* `secret_key` - (CloudWatch) The secret key to use to access the data source.

## Attributes Reference

The resource exports the following attributes:

* `id` - The opaque unique id assigned to the data source by the Grafana
  server.
