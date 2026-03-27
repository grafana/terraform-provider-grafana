# install linux-node integration
resource "grafana_cloud_integration" "linux-node" {
  slug = "linux-node"

  configuration {
    configurable_logs {
      logs_disabled = false
    }
    configurable_alerts {
      alerts_disabled = false
    }
  }
}

# install kafka integration w. alerts disabled
resource "grafana_cloud_integration" "kafka" {
  slug = "kafka"

  configuration {
    configurable_logs {
      logs_disabled = false
    }
    configurable_alerts {
      alerts_disabled = true
    }
  }
}

# Output info
output "linux_node_integration" {
  value = {
    name              = grafana_cloud_integration.linux-node.name
    latest_version    = grafana_cloud_integration.linux-node.latest_version
    installed_version = grafana_cloud_integration.linux-node.installed_version
    dashboard_folder  = grafana_cloud_integration.linux-node.dashboard_folder
  }
}
output "kafka_integration" {
  value = {
    name              = grafana_cloud_integration.kafka.name
    latest_version    = grafana_cloud_integration.kafka.latest_version
    installed_version = grafana_cloud_integration.kafka.installed_version
    dashboard_folder  = grafana_cloud_integration.kafka.dashboard_folder
  }
}
