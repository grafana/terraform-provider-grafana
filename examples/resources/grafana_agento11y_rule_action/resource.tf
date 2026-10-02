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
  rule_id       = "score_user_turns"
  selector      = "user_visible_turn"
  sample_rate   = 0.1
  evaluator_ids = [grafana_agento11y_evaluator.example.evaluator_id]
}

resource "grafana_agento11y_collection" "failed" {
  name        = "Failed evaluations"
  description = "Conversations where every evaluator failed."
}

# Adds conversations to a collection when every evaluator on the rule fails.
resource "grafana_agento11y_rule_action" "example" {
  rule_id        = grafana_agento11y_evaluation_rule.example.rule_id
  condition      = "all_evaluators_fail"
  collection_ids = [grafana_agento11y_collection.failed.id]
}
