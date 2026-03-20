# This is used to test that we can update _acc_basic.tf
resource "grafana_dashboard" "test" {
  config_json = jsonencode({
    title = "Updated Title"
    uid   = "basic"
  })
}
