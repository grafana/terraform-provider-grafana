package provider_test

import (
	"fmt"
	"testing"

	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

// lastProtocolV5Release is the last released provider that served protocol v5.
const lastProtocolV5Release = "4.52.0"

// TestAccProtocolV6Upgrade applies resources with the last protocol v5 release, then checks that
// this build, which serves protocol v6, plans no changes for the same state and can apply it.
// It demonstrates the protocol move and can be removed once that release is well behind us.
//
// The first step downloads the release from the registry. A dev_overrides entry for
// grafana/grafana in your Terraform CLI config replaces it with your local build, so run this
// with TF_CLI_CONFIG_FILE pointing at a config without one.
func TestAccProtocolV6Upgrade(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t, ">=11.0.0")

	for _, tc := range []struct {
		name     string
		versions string
		config   func(suffix string) string
	}{
		{name: "sdkv2_and_framework", versions: ">=11.0.0", config: testAccProtocolV6UpgradeCoreConfig},
		{name: "app_platform_playlist", versions: ">=12.4.0", config: testAccProtocolV6UpgradePlaylistConfig}, // playlist v1 API returns 404 before 12.4
		{name: "app_platform_alertrule", versions: ">=13.1.0", config: testAccProtocolV6UpgradeAlertRuleConfig},
	} {
		t.Run(tc.name, func(t *testing.T) {
			testutils.CheckOSSTestsEnabled(t, tc.versions)

			config := tc.config(acctest.RandString(6))
			resource.Test(t, resource.TestCase{
				Steps: []resource.TestStep{
					{
						ExternalProviders: map[string]resource.ExternalProvider{
							"grafana": {Source: "grafana/grafana", VersionConstraint: lastProtocolV5Release},
						},
						Config: config,
					},
					{
						ProtoV6ProviderFactories: testutils.ProtoV6ProviderFactories,
						Config:                   config,
						PlanOnly:                 true,
					},
					{
						ProtoV6ProviderFactories: testutils.ProtoV6ProviderFactories,
						Config:                   config,
					},
				},
			})
		})
	}
}

func testAccProtocolV6UpgradeCoreConfig(suffix string) string {
	return fmt.Sprintf(`
resource "grafana_folder" "test" {
  title = "upgrade-folder-%[1]s"
}

resource "grafana_team" "test" {
  name = "upgrade-team-%[1]s"
  preferences {
    theme = "dark"
  }
}

resource "grafana_data_source" "test" {
  type              = "prometheus"
  name              = "upgrade-ds-%[1]s"
  url               = "http://localhost:9090"
  json_data_encoded = jsonencode({ httpMethod = "POST" })
}

resource "grafana_dashboard" "test" {
  folder      = grafana_folder.test.uid
  config_json = jsonencode({ title = "upgrade-dash-%[1]s", uid = "upgrade-dash-%[1]s" })
}

resource "grafana_rule_group" "test" {
  name             = "upgrade-group-%[1]s"
  folder_uid       = grafana_folder.test.uid
  interval_seconds = 60
  rule {
    name      = "upgrade-rule"
    condition = "B"
    data {
      ref_id         = "A"
      datasource_uid = grafana_data_source.test.uid
      relative_time_range {
        from = 600
        to   = 0
      }
      model = jsonencode({ expr = "up", refId = "A" })
    }
    data {
      ref_id         = "B"
      datasource_uid = "__expr__"
      relative_time_range {
        from = 0
        to   = 0
      }
      model = jsonencode({ expression = "A", type = "threshold", refId = "B", conditions = [{ evaluator = { params = [1], type = "gt" } }] })
    }
  }
}
`, suffix)
}

func testAccProtocolV6UpgradePlaylistConfig(suffix string) string {
	return fmt.Sprintf(`
resource "grafana_dashboard" "test" {
  config_json = jsonencode({ title = "upgrade-playlist-dash-%[1]s", uid = "upgrade-pl-dash-%[1]s" })
}

resource "grafana_apps_playlist_playlist_v1" "test" {
  metadata {
    uid = "upgrade-playlist-%[1]s"
  }
  spec {
    title    = "upgrade-playlist-%[1]s"
    interval = "5m"
    items = [
      {
        type  = "dashboard_by_uid"
        value = grafana_dashboard.test.uid
      }
    ]
  }
}
`, suffix)
}

func testAccProtocolV6UpgradeAlertRuleConfig(suffix string) string {
	return fmt.Sprintf(`
resource "grafana_folder" "test" {
  title = "upgrade-alert-folder-%[1]s"
}

resource "grafana_data_source" "test" {
  type = "prometheus"
  name = "upgrade-alert-ds-%[1]s"
  url  = "http://localhost:9090"
}

resource "grafana_contact_point" "test" {
  name = "upgrade-alert-cp-%[1]s"
  email {
    addresses = ["test@example.com"]
  }
}

resource "grafana_apps_rules_alertrule_v0alpha1" "test" {
  metadata {
    uid        = "upgrade-alert-%[1]s"
    folder_uid = grafana_folder.test.uid
  }
  spec {
    title = "upgrade-alert-%[1]s"
    trigger {
      interval = "1m"
    }
    paused         = true
    no_data_state  = "KeepLast"
    exec_err_state = "KeepLast"
    notification_settings {
      simplified_routing {
        contact_point = grafana_contact_point.test.name
      }
    }
    expressions = {
      "A" = jsonencode({
        model               = { expr = "up", refId = "A" }
        datasource_uid      = grafana_data_source.test.uid
        relative_time_range = { from = "600s", to = "0s" }
        query_type          = ""
        source              = true
      })
    }
  }
}
`, suffix)
}
