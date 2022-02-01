# This is used to test that we can update the uid of _acc_basic_update.tf
#
# it a point to ensure explicit changes to `uid` are noticed.
resource "grafana_library_panel" "test" {
  name          = "basic_update_uid"
  folder_id     = 0
  model_json    = jsonencode({
    description = "basic_update_uid",
  })
}
