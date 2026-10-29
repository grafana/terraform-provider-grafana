resource "grafana_incident_severity" "test" {
  label       = "tf-acc-test-severity"
  level       = 5
  description = "Terraform acceptance test severity."
}
