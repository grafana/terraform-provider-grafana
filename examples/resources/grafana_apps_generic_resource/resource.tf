# Folder from a YAML manifest file.
resource "grafana_apps_generic_resource" "folder_from_yaml" {
  manifest = yamldecode(file("${path.module}/folder.yaml"))
}

# Folder with an inline manifest.
resource "grafana_apps_generic_resource" "folder_inline" {
  manifest = {
    apiVersion = "folder.grafana.app/v1"
    kind       = "Folder"
    metadata = {
      name = "my-inline-folder"
    }
    spec = {
      title = "My Inline Folder"
    }
  }
}

# Dashboard with annotations and labels.
resource "grafana_apps_generic_resource" "dashboard" {
  manifest = {
    apiVersion = "dashboard.grafana.app/v2"
    kind       = "Dashboard"
    metadata = {
      name = "my-dashboard"
      annotations = {
        "grafana.app/folder" = grafana_apps_generic_resource.folder_inline.metadata.uid
      }
      labels = {
        "team" = "platform"
      }
    }
    spec = {
      title      = "My Dashboard"
      cursorSync = "Off"
      preload    = false
      elements   = {}
      layout     = { kind = "GridLayout", spec = { items = [] } }
    }
  }
}
