resource "grafana_asserts_custom_model_rules" "test" {
  name = "test-anything"
  rules {
    entity {
      type = "Whatever"
      name = "Nothing"
      defined_by {
        query = "up{job=\"nothing\"}"
      }
    }
  }
}
