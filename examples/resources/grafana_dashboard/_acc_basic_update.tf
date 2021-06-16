# this is used as an update on the basic resource above
# NOTE: it leaves out id and version, as this is what
# users will do when updating
resource "grafana_dashboard" "test" {
  config_json = <<EOD
{
  "title": "Updated Title",
  "uid": "update"
}
EOD
}
