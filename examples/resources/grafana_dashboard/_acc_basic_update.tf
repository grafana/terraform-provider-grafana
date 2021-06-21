# This is used to test that we can update _acc_basic.tf
resource "grafana_dashboard" "test" {
  config_json = <<EOD
{
  "title": "Updated Title",
  "uid": "basic"
}
EOD
}
