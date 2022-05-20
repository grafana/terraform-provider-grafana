resource "grafana_organization" "test" {
  name = "test-org"
}

data "grafana_organization" "from_name" {
  name = grafana_organization.test.name
}
