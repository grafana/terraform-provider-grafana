resource "grafana_apps_provisioning_repository_v0alpha1" "local_repo" {
  metadata {
    uid = "my-local-folder-repo"
  }

  spec {
    title       = "My Local Folder Repository"
    description = "Folder-scoped local filesystem repository"
    type        = "local"

    workflows = ["write"]

    sync {
      enabled          = true
      target           = "folder"
      interval_seconds = 60
    }

    local {
      path = "/usr/share/grafana/conf/provisioning/my-local-repo"
    }
  }
}
