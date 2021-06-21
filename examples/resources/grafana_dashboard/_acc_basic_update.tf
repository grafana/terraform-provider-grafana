# This is used to test updates on _acc_basic.tf
resource "grafana_dashboard" "test" {
  config_json = <<EOD
{
  "title": "Updated Title",
  "uid": "update"
}
EOD
}
