# Creating a library panel with a random uid.
# We'd like to ensure that using a computed configuration works.

resource "grafana_library_panel" "test" {
  config_json = <<EOD
{
  "title": "Terraform Acceptance Test"
}
EOD
}

resource "grafana_library_panel" "test-computed" {
  config_json = <<EOD
{
  "title": "Terraform Acceptance Test Computed",
	"tags": ["${grafana_library_panel.test.uid}"]
}
EOD
}
