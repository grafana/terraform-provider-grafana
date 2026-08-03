package assistant_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"

	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
)

func TestAccAssistantAutomation_basic(t *testing.T) {
	testutils.CheckAssistantTestsEnabled(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testutils.TestAccExample(t, "resources/grafana_assistant_automation/_acc_basic.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("grafana_assistant_automation.test", "id"),
					resource.TestCheckResourceAttr("grafana_assistant_automation.test", "name", "tf-acc-test-automation"),
					resource.TestCheckResourceAttr("grafana_assistant_automation.test", "scope", "tenant"),
					resource.TestCheckResourceAttr("grafana_assistant_automation.test", "enabled", "false"),
					resource.TestCheckResourceAttr("grafana_assistant_automation.test", "schedule_cron", "0 9 * * *"),
					resource.TestCheckResourceAttr("grafana_assistant_automation.test", "schedule_timezone", "UTC"),
					testutils.CheckLister("grafana_assistant_automation.test"),
				),
			},
			{
				ResourceName:      "grafana_assistant_automation.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testutils.TestAccExampleWithReplace(t, "resources/grafana_assistant_automation/_acc_basic.tf", map[string]string{
					"tf-acc-test-automation": "tf-acc-test-automation-updated",
				}),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("grafana_assistant_automation.test", "name", "tf-acc-test-automation-updated"),
				),
			},
		},
	})
}
