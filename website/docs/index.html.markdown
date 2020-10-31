---
layout: "grafana"
page_title: "Provider: Grafana"
sidebar_current: "docs-grafana-index"
description: |-
  The Grafana provider configures data sources and dashboards in Grafana.
---

# Grafana Provider

The Grafana provider configures data sources and dashboards in
[Grafana](http://grafana.org/), which is a web application for creating,
viewing and sharing metrics dashboards.

The provider configuration block accepts the following arguments:

* `url` - (Required) The root URL of a Grafana server. May alternatively be set
  via the `GRAFANA_URL` environment variable.

* `auth` - (Required) The API token or username/password to use to authenticate
  to the Grafana server. If username/password is used, they are provided in a
  single string and separated by a colon. May alternatively be set via the
  ``GRAFANA_AUTH`` environment variable.

* org_id - (Required) The organization id to operate on within grafana.
  Default org_id is 1.  May alternatively be set via the
  GRAFANA_ORG_ID environment variable.

* ``tls_key`` - (Optional) Client TLS key file to use to authenticate to the
  Grafana server. May alternatively be set via the ``GRAFANA_TLS_KEY``
  environment variable.

* ``tls_cert`` - (Optional) Client TLS certificate file to use to authenticate
  to the Grafana server. May alternatively be set via the ``GRAFANA_TLS_CERT``
  environment variable.

* ``ca_cert`` - (Optional) Certificate CA bundle to use to verify the
  Grafana server's certifiate.. May alternatively be set via the
  ``GRAFANA_CA_CERT`` environment variable.

* ``insecure_skip_verify`` - (Optional) Bool. Disable verification of the
  Grafana Server's certificate. May alternatively be set via the
  ``GRAFANA_INSECURE_SKIP_VERIFY`` environment variable.

Use the navigation to the left to read about the available resources.

## Example Usage

```hcl
provider "grafana" {
  url    = "http://grafana.example.com/"
  auth   = "1234abcd"
  org_id = 1
}

resource "grafana_dashboard" "metrics" {
  config_json = file("grafana-dashboard.json")
}

resource "grafana_data_source" "influxdb" {
  type          = "influxdb"
  name          = "test_influxdb"
  url           = "http://influxdb.example.net:8086/"
  username      = "foo"
  password      = "bar"
  database_name = "mydb"
}

resource "grafana_alert_notification" "slack" {
  name = "My Slack"
  type = "slack"

  settings = {
    url         = "https://hooks.slack.com/hoook"
    recipient   = "@someguy"
    uploadImage = "false"
  }
}

resource "grafana_organization" "org" {
  name         = "Grafana Organization"
  admin_user   = "admin"
  create_users = true
  admins = [
    "admin@example.com",
  ]
  editors = [
    "editor-01@example.com",
    "editor-02@example.com",
  ]
  viewers = [
    "viewer-01@example.com",
    "viewer-02@example.com",
  ]
}

resource "grafana_team" "team1" {
  name  = "Grafana Team 1"
  email = "GrafanaTeam1@example.com"
  members = [
    "viewer-01@example.com",
  ]
}
```
