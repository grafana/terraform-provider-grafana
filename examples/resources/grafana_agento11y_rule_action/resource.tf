# Adds conversations to a collection when every evaluator on the rule fails.
# The referenced collection must already exist in Agent Observability.
resource "grafana_agento11y_rule_action" "example" {
  rule_id        = grafana_agento11y_evaluation_rule.example.rule_id
  condition      = "all_evaluators_fail"
  collection_ids = ["failed-evaluations"]
}
