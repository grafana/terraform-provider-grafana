data "grafana_synthetic_monitoring_probes" "main" {}

resource "grafana_synthetic_monitoring_check" "grpc" {
  job     = "gRPC Defaults"
  target  = "host.docker.internal:50051"
  enabled = false
  probes = [
    data.grafana_synthetic_monitoring_probes.main.probes.Ohio,
  ]
  labels = {
    foo = "bar"
  }
  settings {
    grpc {}
  }
}
