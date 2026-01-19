resource "grafana_asserts_profile_config" "production" {
  name            = "production"
  priority        = 1000
  default_config  = false
  data_source_uid = "grafanacloud-profiles"

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

  entity_property_to_profile_label_mapping = {
    "cluster"        = "k8s_cluster_name"
    "namespace"      = "k8s_namespace_name"
    "container"      = "k8s_container_name"
    "otel_service"   = "service_name"
    "otel_namespace" = "service_namespace"
  }
}

resource "grafana_asserts_profile_config" "development" {
  name            = "development"
  priority        = 2000
  default_config  = false
  data_source_uid = "grafanacloud-profiles"

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

  entity_property_to_profile_label_mapping = {
    "cluster"        = "k8s_cluster_name"
    "namespace"      = "k8s_namespace_name"
    "container"      = "k8s_container_name"
    "otel_service"   = "service_name"
    "otel_namespace" = "service_namespace"
    "pod"            = "k8s_pod_name"
  }
}

resource "grafana_asserts_profile_config" "minimal" {
  name            = "minimal"
  priority        = 3000
  data_source_uid = "pyroscope-minimal"

  match {
    property = "asserts_entity_type"
    op       = "IS NOT NULL"
  }

  entity_property_to_profile_label_mapping = {
    "cluster"        = "k8s_cluster_name"
    "otel_service"   = "service_name"
    "otel_namespace" = "service_namespace"
  }
}
