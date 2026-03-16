resource "grafana_apps_provisioning_repository_v0alpha1" "github_token" {
  metadata {
    uid = "my-github-folder-repo"
  }

  spec {
    title       = "My GitHub Folder Repository"
    description = "Folder-scoped GitHub repository authenticated directly with a token"
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
  }

  secure {
    token = {
      create = "replace-me"
    }
  }
  secure_version = 1
}
