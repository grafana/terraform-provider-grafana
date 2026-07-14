resource "grafana_data_source" "loki" {
  type = "loki"
  name = "loki"
  url  = "http://localhost:3100"
}

resource "grafana_data_source_cache_config" "loki_cache" {
  datasource_uid   = grafana_data_source.loki.uid
  enabled          = true
  use_default_ttl  = false
  ttl_queries_ms   = 60000
  ttl_resources_ms = 300000
}


