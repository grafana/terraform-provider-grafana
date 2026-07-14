resource "grafana_apps_provisioning_connection_v0alpha1" "github_app" {
  metadata {
    uid = "my-github-app-connection"
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

resource "grafana_apps_provisioning_repository_v0alpha1" "github_app" {
  depends_on = [grafana_apps_provisioning_connection_v0alpha1.github_app]

  metadata {
    uid = "my-github-app-folder-repo"
  }

  spec {
    title       = "My GitHub App Folder Repository"
    description = "Folder-scoped GitHub repository authenticated via a referenced GitHub App connection"
    type        = "github"

    workflows = ["write", "branch"]

    sync {
      enabled          = true
      target           = "folder"
      interval_seconds = 60
    }

    github {
      url    = "https://github.com/example/grafana-dashboards"
      branch = "main"
      path   = "grafanatftest"
    }

    connection {
      name = grafana_apps_provisioning_connection_v0alpha1.github_app.metadata.uid
    }
  }
}
