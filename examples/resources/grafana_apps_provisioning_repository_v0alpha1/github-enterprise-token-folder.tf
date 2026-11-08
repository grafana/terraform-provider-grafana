resource "grafana_apps_provisioning_repository_v0alpha1" "github_enterprise_token" {
  metadata {
    uid = "my-ghes-folder-repo"
  }

  spec {
    title       = "My GitHub Enterprise Folder Repository"
    description = "Folder-scoped GitHub Enterprise Server repository authenticated with a token"
    type        = "githubEnterprise"

    workflows = ["write", "branch"]

    sync {
      enabled          = true
      target           = "folder"
      interval_seconds = 60
    }

    github_enterprise {
      server_url = "https://ghes.example.com"
      url        = "https://ghes.example.com/example/grafana-dashboards"
      branch     = "main"
      path       = "grafanatftest"
    }

    branch {
      name_template    = "grafana/{{title}}-{{random}}"
      enforce_template = true
    }

    pull_request {
      title_template   = "Update {{title}}"
      enforce_template = false
    }

    commit {
      single_resource_message_template = "Save {{resourceKind}}: {{title}}"
      enforce_template                 = true
      signer_name                      = "Grafana Bot"
      signer_email                     = "bot@example.com"
      signing_method                   = "gpg"
    }
  }

  secure {
    token = {
      create = "replace-me"
    }
    commit_signing_key = {
      create = "replace-me-with-private-key"
    }
  }
  secure_version = 1
}
