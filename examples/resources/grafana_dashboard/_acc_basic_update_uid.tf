# This is used to test that we can update the uid of _acc_basic_update.tf
#
# Since uid is tracked separately, changes to uid must trigger an update.
resource "grafana_dashboard" "test" {
  config_json = jsonencode({
    title = "Updated Title"
    uid   = "basic-update"
  })
}
