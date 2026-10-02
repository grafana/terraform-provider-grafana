resource "grafana_assistant_quickstart" "test" {
  scope  = "tenant"
  title  = "tf-acc-test-quickstart"
  prompt = "How healthy are my SLOs right now?"
}
