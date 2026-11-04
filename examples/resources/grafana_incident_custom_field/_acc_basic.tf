resource "grafana_incident_custom_field" "test" {
  name        = "tf-acc-test-field"
  slug        = "tf_acc_test_field"
  type        = "string"
  description = "Terraform acceptance test field."
}
