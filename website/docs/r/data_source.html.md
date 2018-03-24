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
source selected (via the `type` argument). The following example is for
InfluxDB. See
[Grafana's *Data Sources Guides*](http://docs.grafana.org/#data-sources-guides)
for more details on the supported data source types and the arguments they use.

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
  application will access the data source. The default is "proxy", which means
  that the application will make requests via a proxy endpoint on the Grafana
  server.

JSON Data (`json_data`) supports the following:

* `auth_type` - (Required by some data source types) The authentication type
  type used to access the data source.

* `default` - (Required by some data source types) The default region for
  the data source.

Secure JSON Data (`secure_json_data`) supports the following:

* `access_key` - (Required by some data source types) The access key required
  to access the data source.

* `secret_key` - (Required by some data source types) The secret key required
  to access the data source.

## Attributes Reference

The resource exports the following attributes:

* `id` - The opaque unique id assigned to the data source by the Grafana
  server.
