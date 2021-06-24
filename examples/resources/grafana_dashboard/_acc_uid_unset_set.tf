# Adding a uid to dashboard that did not previously have one.
# We'd like to ensure that adding a uid causes the resource to update.
resource "grafana_dashboard" "test" {
  config_json = <<EOD
{
  "title": "UID Unset",
  "uid": "uid-previously-unset"
}
EOD
}
