resource "grafana_incident_role" "test" {
  name        = "tf-acc-test-role"
  description = "Terraform acceptance test role."
  archived    = true
}
