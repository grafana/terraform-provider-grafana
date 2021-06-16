# The "id" and "version" properties in the config below are there to test that
# we correctly normalize them away. They are not actually used by this
# resource, since it uses slugs for identification.
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
}
