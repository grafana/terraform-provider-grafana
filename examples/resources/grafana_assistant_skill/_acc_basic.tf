resource "grafana_assistant_skill" "test" {
  name                     = "tf-acc-test-skill"
  body                     = "Terraform acceptance test skill body."
  scope                    = "tenant"
  include_in_knowledgebase = true
}
