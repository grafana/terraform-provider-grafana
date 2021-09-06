# This is used to test that we can update the uid of _acc_basic_update.tf
#
# Since uid is removed from `config_json` before writing it to state, we must make
# it a point to ensure explicit changes to `uid` are noticed.
resource "grafana_dashboard" "test" {
  config_json = <<EOD
{
  "title": "Updated Title",
  "uid": "basic-update"
}
EOD
}
