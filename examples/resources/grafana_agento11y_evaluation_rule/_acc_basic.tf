resource "grafana_agento11y_evaluator" "test" {
  evaluator_id = "tf_acc_test_rule_evaluator"
  version      = "1"
  kind         = "regex"

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

resource "grafana_agento11y_evaluation_rule" "test" {
  rule_id       = "tf_acc_test_rule"
  enabled       = true
  selector      = "user_visible_turn"
  sample_rate   = 0.1
  evaluator_ids = [grafana_agento11y_evaluator.test.evaluator_id]
}
