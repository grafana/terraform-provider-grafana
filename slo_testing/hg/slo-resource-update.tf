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
  name        = "Hello1"
  description = "Testing Hello 1 - I hope this works!"
  service     = "service5"
  query       = "sum(rate(apiserver_request_total{code!=\"300\"}[$__rate_interval])) / sum(rate(apiserver_request_total[$__rate_interval]))"
  objectives {
    objective_value  = 0.95
    objective_window = "28d"
  }
  labels {
    key   = "label1a"
    value = "value1a"
  }
  labels {
    key   = "label2a"
    value = "value2a"
  }
  alerting {
    name = "hihialerting1"
    labels {
      key   = "alertinglabel1"
      value = "alertingvalue1"
    }

    annotations {
      key   = "alertingannot1"
      value = "alertingvalue1"
    }

    fastburn {
      labels {
        key   = "labelsfastburnkey1"
        value = "labelsfastburnvalue1"
      }
      annotations {
        key   = "annotsfastburnannot1"
        value = "annotsfastburnvalue1"
      }
    }

    slowburn {
      labels {
        key   = "labelsslowburnkey1"
        value = "labelsslowburnvalue1"
      }
      annotations {
        key   = "annotsslowburnannot2"
        value = "annotsslowburnvalue1"
      }
    }
  }
}

output "test1_order" {
  value = grafana_slo_resource.test1
}