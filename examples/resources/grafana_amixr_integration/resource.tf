resource "grafana_amixr_integration" "test-acc-integration" {
  provider = grafana.amixr
  name     = "my integration"
  type     = "grafana"
  default_route {
  }
}
