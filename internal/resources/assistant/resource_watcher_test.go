package assistant_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"

	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
)

func TestAccAssistantWatcher_basic(t *testing.T) {
	testutils.CheckAssistantTestsEnabled(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testutils.TestAccExample(t, "resources/grafana_assistant_watcher/_acc_basic.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("grafana_assistant_watcher.test", "id"),
					resource.TestCheckResourceAttr("grafana_assistant_watcher.test", "name", "tf-acc-test-watcher"),
					resource.TestCheckResourceAttr("grafana_assistant_watcher.test", "sensitivity", "balanced"),
					resource.TestCheckResourceAttr("grafana_assistant_watcher.test", "trigger_interval_seconds", "900"),
					resource.TestCheckResourceAttr("grafana_assistant_watcher.test", "query.#", "1"),
					resource.TestCheckResourceAttr("grafana_assistant_watcher.test", "query.0.type", "alerts"),
					// Supplying the calibrated checks moves the watcher out of
					// draft without an interactive calibration session.
					resource.TestCheckResourceAttr("grafana_assistant_watcher.test", "status", "ready"),
					resource.TestCheckResourceAttr("grafana_assistant_watcher.test", "started", "false"),
					resource.TestCheckResourceAttrSet("grafana_assistant_watcher.test", "calibrated_at"),
					testutils.CheckLister("grafana_assistant_watcher.test"),
				),
			},
			{
				ResourceName:      "grafana_assistant_watcher.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testutils.TestAccExampleWithReplace(t, "resources/grafana_assistant_watcher/_acc_basic.tf", map[string]string{
					"tf-acc-test-watcher": "tf-acc-test-watcher-updated",
				}),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("grafana_assistant_watcher.test", "name", "tf-acc-test-watcher-updated"),
					resource.TestCheckResourceAttr("grafana_assistant_watcher.test", "status", "ready"),
				),
			},
		},
	})
}
