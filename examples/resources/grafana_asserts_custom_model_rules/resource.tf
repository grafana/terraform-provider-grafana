resource "grafana_asserts_custom_model_rules" "test" {
  name = "test-anything"
  rules {
    entity {
      type = "Service"
      name = "workload | service | job"
      scope = {
        namespace = "namespace"
        env       = "asserts_env"
        site      = "asserts_site"
      }
      lookup = {
        workload  = "workload | deployment | statefulset | daemonset | replicaset"
        service   = "service"
        job       = "job"
        proxy_job = "job"
      }
      defined_by {
        query = "up{job!=''}"
        disabled = false
        label_values = {
          service = "service"
          job     = "job"
        }
        literals = {
          _source = "up_query"
        }
      }
      defined_by {
        query = "up{job='disabled'}"
        disabled = true
      }
    }
  }
}
