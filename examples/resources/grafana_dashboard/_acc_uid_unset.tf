# Test that we don't get a diff when uid is unset.
# In this case, it will be generated by Grafana.
resource "grafana_dashboard" "test" {
  config_json = <<EOD
{
  "title": "UID Unset"
}
EOD
}

