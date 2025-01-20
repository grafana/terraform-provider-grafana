data "grafana_cloud_stack" "current" {
  slug = "<your slug>"
}

resource "grafana_cloud_private_data_source_connect_network" "test" {
  region           = "us"
  name             = "my-pdc"
  display_name     = "My PDC"
  stack_identifier = data.grafana_cloud_stack.current.id
}

resource "grafana_cloud_private_data_source_connect_network_token" "test" {
  pdc_network_id = grafana_cloud_private_data_source_connect_network.test.pdc_network_id
  region         = grafana_cloud_private_data_source_connect_network.test.region
  name           = "my-pdc-token"
  display_name   = "My PDC Token"
}
