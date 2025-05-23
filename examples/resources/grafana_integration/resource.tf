# Install Docker integration with logs and alerts enabled
resource "grafana_integration" "docker" {
  slug = "docker"
  
  configuration {
    configurable_logs {
      logs_disabled = false
    }
    configurable_alerts {
      alerts_disabled = false
    }
  }
}

# Install Linux Node integration with logs enabled but alerts disabled
resource "grafana_integration" "linux_node" {
  slug = "linux-node"
  
  configuration {
    configurable_logs {
      logs_disabled = false
    }
    configurable_alerts {
      alerts_disabled = true
    }
  }
}

# Install Windows integration with minimal configuration
resource "grafana_integration" "windows" {
  slug = "windows-exporter"
}

# Output integration information
output "docker_integration" {
  value = {
    name              = grafana_integration.docker.name
    version           = grafana_integration.docker.version
    installed         = grafana_integration.docker.installed
    dashboard_folder  = grafana_integration.docker.dashboard_folder
  }
}
