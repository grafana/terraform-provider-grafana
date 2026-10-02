resource "grafana_agento11y_hook_rule" "example" {
  rule_id        = "block_destructive_tools"
  phase          = "preflight"
  action_on_fail = "deny"

  blocked_tools = ["delete_*", "drop_*"]

  redact {
    id    = "emails"
    regex = "[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+"
  }
}
