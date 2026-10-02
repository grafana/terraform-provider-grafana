resource "grafana_agento11y_collection" "test" {
  name        = "tf_acc_test_collection"
  description = "Managed by the Terraform provider acceptance tests."
}
