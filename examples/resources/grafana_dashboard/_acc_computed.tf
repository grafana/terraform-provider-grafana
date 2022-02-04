# Creating a dashboard with a random uid.
# We'd like to ensure that using a computed configuration works.

resource "grafana_dashboard" "test" {
  config_json = <<EOD
{
  "title": "Terraform Acceptance Test"
}
EOD
}

resource "grafana_dashboard" "test-computed" {
  config_json = <<EOD
{
  "title": "Terraform Acceptance Test Computed",
	"tags": ["${grafana_dashboard.test.uid}"]
}
EOD
}
