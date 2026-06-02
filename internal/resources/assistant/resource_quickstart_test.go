package assistant_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"

	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
)

func TestAccAssistantQuickstart_basic(t *testing.T) {
	testutils.CheckAssistantTestsEnabled(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testutils.TestAccExample(t, "resources/grafana_assistant_quickstart/_acc_basic.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("grafana_assistant_quickstart.test", "id"),
					resource.TestCheckResourceAttr("grafana_assistant_quickstart.test", "title", "tf-acc-test-quickstart"),
					resource.TestCheckResourceAttr("grafana_assistant_quickstart.test", "scope", "tenant"),
					resource.TestCheckResourceAttr("grafana_assistant_quickstart.test", "prompt", "How healthy are my SLOs right now?"),
					testutils.CheckLister("grafana_assistant_quickstart.test"),
				),
			},
			{
				ResourceName:      "grafana_assistant_quickstart.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testutils.TestAccExampleWithReplace(t, "resources/grafana_assistant_quickstart/_acc_basic.tf", map[string]string{
					"tf-acc-test-quickstart": "tf-acc-test-quickstart-updated",
				}),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("grafana_assistant_quickstart.test", "title", "tf-acc-test-quickstart-updated"),
				),
			},
		},
	})
}
