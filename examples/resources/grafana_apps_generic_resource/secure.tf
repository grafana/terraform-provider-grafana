# Repository with secure fields.
# Secure values are top-level because `secure` is write-only and cannot live in the manifest.
resource "grafana_apps_generic_resource" "repository" {
  manifest = {
    apiVersion = "provisioning.grafana.app/v1beta1"
    kind       = "Repository"
    metadata = {
      name = "platform-repo"
    }
    spec = {
      title       = "Platform Repository"
      description = "Repository managed through the generic resource"
      type        = "github"
      workflows   = ["write"]
      sync = {
        enabled         = false
        target          = "instance"
        intervalSeconds = 300
      }
      github = {
        url                       = "https://github.com/example/grafana-dashboards"
        branch                    = "main"
        path                      = "grafana"
        generateDashboardPreviews = false
      }
    }
  }

  secure = {
    token         = { create = var.github_token }
    webhookSecret = { create = var.webhook_secret }
  }
  secure_version = 1
}

# Example import:
# terraform import grafana_apps_generic_resource.repository provisioning.grafana.app/v1beta1/Repository/platform-repo
# After import, add `secure` and `secure_version` back manually because write-only arguments are not stored in state.
