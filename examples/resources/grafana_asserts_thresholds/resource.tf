resource "grafana_asserts_thresholds" "basic" {
  request_thresholds = [{
    entity_name     = "payment-service"
    assertion_name  = "ErrorRatioBreach"
    request_type    = "inbound"
    request_context = "/charge"
    value           = 0.01
  }]

  resource_thresholds = [{
    assertion_name = "Saturation"
    resource_type  = "container"
    container_name = "worker"
    source         = "metrics"
    severity       = "warning"
    value          = 75
  }]

  health_thresholds = [{
    assertion_name = "ServiceDown"
    expression     = "up < 1"
    entity_type    = "Service"
  }]
}
