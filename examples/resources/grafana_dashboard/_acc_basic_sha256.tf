resource "grafana_dashboard" "test_sha256" {
  config_json = <<EOD
{
  "title": "Terraform Acceptance Test",
  "uid": "basic"
}
EOD
}
