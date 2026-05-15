resource "grafana_apps_provisioning_repository_v0alpha1" "gitlab_token" {
  metadata {
    uid = "my-gitlab-folder-repo"
  }

  spec {
    title       = "My GitLab Folder Repository"
    description = "Folder-scoped GitLab repository authenticated directly with a token"
    type        = "gitlab"

    workflows = ["write", "branch"]

    sync {
      enabled          = true
      target           = "folder"
      interval_seconds = 60
    }

    gitlab {
      url    = "https://gitlab.com/example/grafana-dashboards"
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
