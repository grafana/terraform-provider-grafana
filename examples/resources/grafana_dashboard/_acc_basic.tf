# The "id" and "version" properties in the config below are there to test that
# we correctly remove them from config_json and manage them in dedicated,
# computed fields.
#
resource "grafana_dashboard" "test" {
  config_json = <<EOD
{
  "title": "Terraform Acceptance Test",
  "id": 12,
  "uid": "basic",
  "version": 34
}
EOD
  message     = "inital commit."
}
