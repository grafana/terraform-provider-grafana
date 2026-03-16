resource "grafana_asserts_trace_config" "production" {
  name            = "production"
  priority        = 1000
  default_config  = false
  data_source_uid = "grafanacloud-traces"

  match {
    property = "asserts_entity_type"
    op       = "="
    values   = ["Service"]
  }

  match {
    property = "deployment_environment"
    op       = "="
    values   = ["production", "staging"]
  }

  match {
    property = "asserts_site"
    op       = "="
    values   = ["us-east-1", "us-west-2"]
  }

  entity_property_to_trace_label_mapping = {
    "cluster"        = "resource.k8s.cluster.name"
    "namespace"      = "resource.k8s.namespace"
    "container"      = "resource.container.name"
    "otel_service"   = "resource.service.name"
    "otel_namespace" = "resource.service.namespace"
  }
}

resource "grafana_asserts_trace_config" "development" {
  name            = "development"
  priority        = 2000
  default_config  = false
  data_source_uid = "grafanacloud-traces"

  match {
    property = "asserts_entity_type"
    op       = "="
    values   = ["Service"]
  }

  match {
    property = "deployment_environment"
    op       = "="
    values   = ["development", "testing"]
  }

  match {
    property = "asserts_site"
    op       = "="
    values   = ["us-east-1"]
  }

  match {
    property = "service"
    op       = "="
    values   = ["my sample api"]
  }

  entity_property_to_trace_label_mapping = {
    "cluster"        = "resource.k8s.cluster.name"
    "namespace"      = "resource.k8s.namespace"
    "container"      = "resource.container.name"
    "otel_service"   = "resource.service.name"
    "otel_namespace" = "resource.service.namespace"
    "pod"            = "span.k8s.pod.name"
  }
}

resource "grafana_asserts_trace_config" "minimal" {
  name            = "minimal"
  priority        = 3000
  data_source_uid = "tempo-minimal"

  match {
    property = "asserts_entity_type"
    op       = "IS NOT NULL"
  }

  entity_property_to_trace_label_mapping = {
    "cluster"        = "resource.k8s.cluster.name"
    "otel_service"   = "resource.service.name"
    "otel_namespace" = "resource.service.namespace"
  }
}
