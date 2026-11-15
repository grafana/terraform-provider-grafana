resource "grafana_incident_custom_field" "test" {
  name = "tf-acc-test-select"
  slug = "tf_acc_test_select"
  type = "single-select"

  select_option {
    value = "low"
    label = "Low"
  }

  select_option {
    value       = "high"
    label       = "High"
    description = "Terraform acceptance test option."
  }
}
