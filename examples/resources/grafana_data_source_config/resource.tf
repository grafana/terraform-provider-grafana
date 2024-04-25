resource "grafana_data_source" "loki" {
  type = "loki"
  name = "loki"
  url  = "http://localhost:3100"

  lifecycle {
    ignore_changes = [json_data_encoded, http_headers]
  }
}

resource "grafana_data_source" "tempo" {
  type = "tempo"
  name = "tempo"
  url  = "http://localhost:3200"

  lifecycle {
    ignore_changes = [json_data_encoded, http_headers]
  }
}

resource "grafana_data_source_config" "loki" {
  uid = grafana_data_source.loki.uid

  json_data_encoded = jsonencode({
    derivedFields = [
      {
        datasourceUid = grafana_data_source.tempo.uid
        matcherRegex  = "[tT]race_?[iI][dD]\"?[:=]\"?(\\w+)"
        matcherType   = "regex"
        name          = "traceID"
        url           = "$${__value.raw}"
      }
    ]
  })
}

resource "grafana_data_source_config" "tempo" {
  uid = grafana_data_source.tempo.uid

  json_data_encoded = jsonencode({
    tracesToLogsV2 = {
      customQuery     = true
      datasourceUid   = grafana_data_source.loki.uid
      filterBySpanID  = false
      filterByTraceID = false
      query           = "|=\"$${__trace.traceId}\" | json"
    }
  })
}
