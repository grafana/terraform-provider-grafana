package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFindReferences(t *testing.T) {
	content := `
resource "grafana_rule_group" "rg" {
  folder_uid = grafana_folder.f.uid
  rule {
    annotations = {
      a = "b"
    }
    notification_settings {
      contact_point = grafana_contact_point.cp.name
    }
  }
  org_id = grafana_organization.o.id
}

resource "grafana_apps_provisioning_repository_v0alpha1" "repo" {
  spec {
    connection {
      name = grafana_apps_provisioning_connection_v0alpha1.conn.metadata.uid
    }
  }
}

locals {
  folder = grafana_folder.f.uid
}
`
	require.Equal(t, []string{
		"grafana_rule_group.folder_uid=grafana_folder.uid",
		"grafana_rule_group.rule.notification_settings.contact_point=grafana_contact_point.name",
		"grafana_rule_group.org_id=grafana_organization.id",
		"grafana_apps_provisioning_repository_v0alpha1.spec.connection.name=grafana_apps_provisioning_connection_v0alpha1.metadata.uid",
	}, findReferences(content))
}
