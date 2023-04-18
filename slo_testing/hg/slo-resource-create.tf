terraform {
  required_providers {
    grafana = { 
      version = "0.2"
      source  = "registry.terraform.io/grafana/grafana"
    }
  }
}

provider "grafana" {
  url = "https://elainetest.grafana.net"
}

resource "grafana_slo_resource" "test1" {
  name        = "Terraform1 - 99.5% of Responses from Kubernetes API Server Valid"
  description = "Terraform1 - Measures that 99.5% of responses from the Kubernetes API Server are valid (i.e. not HTTP 500 Errors)"
  service     = "service1"
  query       = "sum(rate(apiserver_request_total{code!=\"500\"}[$__rate_interval])) / sum(rate(apiserver_request_total[$__rate_interval]))"
  objectives {
    objective_value  = 0.995
    objective_window = "30d"
  }
  labels {
    key   = "custom"
    value = "value"
  }
  alerting {
    fastburn {
      annotations {
        key   = "name"
        value = "Critical - SLO Burn Rate Alert"
      }
      annotations {
        key   = "description"
        value = "Error Budget is burning at a rate greater than 14.4x. This means that within 1 Hour, 2% of your SLO Error Budget may be consumed. Recommended action: Page"
      }
      labels {
        key   = "type"
        value = "slo"
      }
    }

    slowburn {
      annotations {
        key   = "name"
        value = "Warning - SLO Burn Rate Alert"
      }
      annotations {
        key   = "description"
        value = "Error Budget is burning at a rate greater than 1x.  This means that within 72 Hours, 10% of your SLO Error Budget may be consumed. Recommended action: Page/Ticket"
      }
      labels {
        key   = "type"
        value = "slo"
      }
    }
  }
}

output "test2_order" {
  value = grafana_slo_resource.test2
}

resource "grafana_slo_resource" "test2" {
  name        = "Terraform2 - 99.5% of Responses from Kubernetes API Server Valid"
  description = "Terraform2 - Measures that 99.5% of responses from the Kubernetes API Server are valid (i.e. not HTTP 500 Errors)"
  service     = "service2"
  query       = "sum(rate(apiserver_request_total{code!=\"500\"}[$__rate_interval])) / sum(rate(apiserver_request_total[$__rate_interval]))"
  objectives {
    objective_value  = 0.995
    objective_window = "30d"
  }
  labels {
    key   = "custom"
    value = "value"
  }
  alerting {
    fastburn {
      annotations {
        key   = "name"
        value = "Critical - SLO Burn Rate Alert"
      }
      annotations {
        key   = "description"
        value = "Error Budget is burning at a rate greater than 14.4x. This means that within 1 Hour, 2% of your SLO Error Budget may be consumed. Recommended action: Page"
      }
      labels {
        key   = "type"
        value = "slo"
      }
    }

    slowburn {
      annotations {
        key   = "name"
        value = "Warning - SLO Burn Rate Alert"
      }
      annotations {
        key   = "description"
        value = "Error Budget is burning at a rate greater than 1x.  This means that within 72 Hours, 10% of your SLO Error Budget may be consumed. Recommended action: Page/Ticket"
      }
      labels {
        key   = "type"
        value = "slo"
      }
    }
  }
}