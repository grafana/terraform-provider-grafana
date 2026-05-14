resource "grafana_apps_provisioning_connection_v0alpha1" "example" {
  metadata {
    uid = "my-github-connection"
  }

  spec {
    title       = "My GitHub App Connection"
    description = "GitHub App connection used by a folder-scoped Git Sync repository"
    type        = "github"
    url         = "https://github.com"

    github {
      app_id          = "12345"
      installation_id = "67890"
    }
  }

  secure {
    private_key = {
      create = filebase64("${path.module}/private-key.pem")
    }
  }
  secure_version = 1
}
