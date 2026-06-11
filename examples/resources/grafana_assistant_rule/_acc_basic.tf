resource "grafana_assistant_rule" "test" {
  name         = "tf-acc-test-rule"
  rule_content = "Terraform acceptance test rule."
  scope        = "tenant"
  priority     = 100
  applications = ["assistant"]
}
