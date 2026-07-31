resource "grafana_agento11y_hook_rule" "test" {
  rule_id        = "tf_acc_test_hook_rule"
  phase          = "preflight"
  action_on_fail = "deny"

  blocked_tools = ["delete_*"]
}
