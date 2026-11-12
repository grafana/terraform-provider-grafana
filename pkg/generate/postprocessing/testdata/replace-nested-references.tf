resource "grafana_contact_point" "cp" {
  name = "my-cp"
}

resource "grafana_rule_group" "rg" {
  name = "my-cp"
  rule {
    name = "first"
  }
  rule {
    name = "second"
    notification_settings {
      contact_point = "my-cp"
    }
  }
}

resource "grafana_apps_provisioning_connection_v0alpha1" "conn" {
  metadata {
    uid = "my-conn"
  }
}

resource "grafana_apps_provisioning_repository_v0alpha1" "repo" {
  spec {
    connection {
      name = "my-conn"
    }
  }
}
