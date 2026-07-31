resource "grafana_agento11y_evaluator" "example" {
  evaluator_id = "no_secrets"
  version      = "1"
  kind         = "regex"
  description  = "Flags assistant responses that leak secrets."

  config = jsonencode({
    pattern = "(?i)(api[_-]?key|password|secret)"
  })

  output_keys = jsonencode([
    {
      key  = "clean"
      type = "bool"
    }
  ])
}
