resource "grafana_synthetic_monitoring_check" "grpc" {
  job                 = "gRPC Defaults"
  target              = "host.docker.internal:50051"
  enabled             = false
  select_probes_count = 1
  labels = {
    foo = "bar"
  }
  settings {
    grpc {}
  }
}
