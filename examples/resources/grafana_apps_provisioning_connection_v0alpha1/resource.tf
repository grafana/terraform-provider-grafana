resource "grafana_apps_provisioning_connection_v0alpha1" "example" {
  metadata {
    uid = "my-github-connection"
  }

  spec {
    title = "My GitHub App Connection"
    type  = "github"
    url   = "https://github.com"

    github {
      app_id          = "12345"
      installation_id = "67890"
    }
  }

  secure {
    private_key = {
      create = "replace-me"
    }
  }
  secure_version = 1
}
