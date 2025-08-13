resource "grafana_asserts_custom_model_rules" "test" {
  name  = "test-anything"
  rules = <<-EOT
    entities:
    - type: Whatever
      name: Nothing
      definedBy:
      - query: "up{job=\\"nothing\\"}"
  EOT
}
