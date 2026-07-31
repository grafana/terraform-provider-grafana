resource "grafana_agento11y_evaluator" "test" {
  evaluator_id = "tf_acc_test_evaluator"
  version      = "1"
  kind         = "regex"
  description  = "Terraform acceptance test evaluator."

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
