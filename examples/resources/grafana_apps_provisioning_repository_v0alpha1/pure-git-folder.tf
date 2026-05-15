resource "grafana_apps_provisioning_repository_v0alpha1" "pure_git" {
  metadata {
    uid = "my-pure-git-folder-repo"
  }

  spec {
    title       = "My Pure Git Folder Repository"
    description = "Folder-scoped generic Git repository authenticated with a token"
    type        = "git"

    workflows = ["write"]

    sync {
      enabled          = true
      target           = "folder"
      interval_seconds = 60
    }

    git {
      url        = "https://git.example.com/platform/dashboards.git"
      branch     = "main"
      path       = "grafanatftest"
      token_user = "git"
    }
  }

  secure {
    token = {
      create = "replace-me"
    }
  }
  secure_version = 1
}
