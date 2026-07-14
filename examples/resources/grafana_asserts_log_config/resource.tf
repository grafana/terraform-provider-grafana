resource "grafana_asserts_log_config" "production" {
  name            = "production"
  priority        = 1000
  default_config  = false
  data_source_uid = "grafanacloud-logs"
  error_label     = "error"

  match {
    property = "asserts_entity_type"
    op       = "="
    values   = ["Service"]
  }

  match {
    property = "environment"
    op       = "="
    values   = ["production", "staging"]
  }

  match {
    property = "site"
    op       = "="
    values   = ["us-east-1", "us-west-2"]
  }

  entity_property_to_log_label_mapping = {
    "otel_namespace" = "service_namespace"
    "otel_service"   = "service_name"
    "environment"    = "env"
    "site"           = "region"
  }

  filter_by_span_id  = true
  filter_by_trace_id = true
}

resource "grafana_asserts_log_config" "development" {
  name            = "development"
  priority        = 2000
  default_config  = true
  data_source_uid = "elasticsearch-dev"
  error_label     = "error"

  match {
    property = "asserts_entity_type"
    op       = "="
    values   = ["Service"]
  }

  match {
    property = "environment"
    op       = "="
    values   = ["development", "testing"]
  }

  match {
    property = "site"
    op       = "="
    values   = ["us-east-1"]
  }

  match {
    property = "service"
    op       = "="
    values   = ["api"]
  }

  entity_property_to_log_label_mapping = {
    "otel_namespace" = "service_namespace"
    "otel_service"   = "service_name"
    "environment"    = "env"
    "site"           = "region"
    "service"        = "app"
  }

  filter_by_span_id  = true
  filter_by_trace_id = true
}

resource "grafana_asserts_log_config" "minimal" {
  name            = "minimal"
  priority        = 3000
  default_config  = false
  data_source_uid = "loki-minimal"

  match {
    property = "asserts_entity_type"
    op       = "IS NOT NULL"
    values   = []
  }
}
