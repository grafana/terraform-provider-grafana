* [Official documentation](https://grafana.com/docs/grafana/latest/datasources/)
* [HTTP API](https://grafana.com/docs/grafana/latest/developers/http_api/data_source/)

The required arguments for this resource vary depending on the type of data
source selected (via the 'type' argument).

Use this resource for configuring multiple datasources, when that configuration (`json_data_encoded` field) requires circular references like in the example below.

> When using the `grafana_data_source_config` resource, the corresponding `grafana_data_source` resources must have the `json_data_encoded` and `http_headers` fields ignored. Otherwise, an infinite update loop will occur. See the example below.
