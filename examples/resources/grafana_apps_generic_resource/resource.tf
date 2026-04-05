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
        "grafana.app/folder" = "my-inline-folder"
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

  depends_on = [grafana_apps_generic_resource.folder_inline]
}

# Inject Terraform variables into a static manifest using merge().
resource "grafana_apps_generic_resource" "folder_with_variable" {
  manifest = merge(yamldecode(file("${path.module}/folder.yaml")), {
    metadata = merge(yamldecode(file("${path.module}/folder.yaml")).metadata, {
      name = "my-dynamic-folder"
    })
    spec = {
      title       = var.folder_title
      description = "A folder managed by Terraform"
    }
  })
}
