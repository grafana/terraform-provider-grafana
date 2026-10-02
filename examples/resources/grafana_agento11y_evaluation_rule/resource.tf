resource "grafana_agento11y_evaluator" "example" {
  evaluator_id = "no_secrets"
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

resource "grafana_agento11y_evaluation_rule" "example" {
  rule_id             = "score_user_turns"
  enabled             = true
  selector            = "user_visible_turn"
  sample_rate         = 0.1
  evaluator_ids       = [grafana_agento11y_evaluator.example.evaluator_id]
  execution_mode      = "parallel"
  filterable_tag_keys = ["environment", "team"]

  match = jsonencode({
    agent_name = "checkout-*"
  })
}
