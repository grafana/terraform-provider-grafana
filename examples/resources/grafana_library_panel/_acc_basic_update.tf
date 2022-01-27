# This is used to test that we can update _acc_basic.tf
resource "grafana_library_panel" "test" {
  name          = "updated name"
  model_json    = jsonencode({
    title       = "updated name",
    id          = 12,
    version     = 35
  })
}
