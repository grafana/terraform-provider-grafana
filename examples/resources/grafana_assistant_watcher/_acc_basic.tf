resource "grafana_assistant_watcher" "test" {
  name        = "tf-acc-test-watcher"
  description = "Terraform acceptance test watcher."
  prompt      = "Watch for the Terraform acceptance test alert firing."

  # An alerts-only watcher needs no data source access, which keeps the
  # acceptance test independent of the datasources on the test stack.
  calibration_context = "Terraform acceptance test watcher. A firing alert is itself the signal."

  query {
    type    = "alerts"
    expr    = "alertname=\"TerraformAccTestAlert\""
    comment = "Firing status of the acceptance test alert."
  }
}
