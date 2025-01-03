data "grafana_cloud_stack" "current" {
  stackID = "<your stack ID>"
}

resource "grafana_cloud_private_datasource_connect" "test" {
  region       = "us"
  name         = "my-pdc"
  display_name = "My PDC"
  identifier   = data.grafana_cloud_stack.current.stackID
}

resource "grafana_cloud_private_datasource_connect_token" "test" {
  pdc_network_id = grafana_cloud_private_datasource_connect.test.network_id
  region         = "us"
  name           = "my-pdc-token"
  display_name   = "My PDC Token"
}
