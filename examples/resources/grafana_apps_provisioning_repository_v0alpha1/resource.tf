resource "grafana_apps_provisioning_repository_v0alpha1" "example" {
  metadata {
    uid = "my-github-repo"
  }

  spec {
    title = "My GitHub Repository"
    type  = "github"

    workflows = ["write"]

    sync {
      enabled          = true
      target           = "folder"
      interval_seconds = 60
    }

    github {
      url    = "https://github.com/example/grafana-dashboards"
      branch = "main"
      path   = "grafana/"
    }
  }

  secure {
    token = {
      create = "replace-me"
    }
  }
  secure_version = 1
}
