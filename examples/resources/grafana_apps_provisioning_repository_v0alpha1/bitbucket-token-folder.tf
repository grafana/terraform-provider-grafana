resource "grafana_apps_provisioning_repository_v0alpha1" "bitbucket_token" {
  metadata {
    uid = "my-bitbucket-folder-repo"
  }

  spec {
    title       = "My Bitbucket Folder Repository"
    description = "Folder-scoped Bitbucket repository authenticated directly with an Atlassian API token"
    type        = "bitbucket"

    workflows = ["write", "branch"]

    sync {
      enabled          = true
      target           = "folder"
      interval_seconds = 60
    }

    bitbucket {
      url        = "https://bitbucket.org/example/grafana-dashboards"
      branch     = "main"
      path       = "grafanatftest"
      token_user = "x-bitbucket-api-token-auth"
    }
  }

  secure {
    token = {
      create = "replace-me"
    }
  }
  secure_version = 1
}
