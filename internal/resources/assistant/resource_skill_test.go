package assistant_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"

	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
)

func TestAccAssistantSkill_basic(t *testing.T) {
	testutils.CheckAssistantTestsEnabled(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testutils.TestAccExample(t, "resources/grafana_assistant_skill/_acc_basic.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("grafana_assistant_skill.test", "id"),
					resource.TestCheckResourceAttr("grafana_assistant_skill.test", "name", "tf-acc-test-skill"),
					resource.TestCheckResourceAttr("grafana_assistant_skill.test", "command_name", "tf-acc-command"),
					resource.TestCheckResourceAttr("grafana_assistant_skill.test", "scope", "tenant"),
					resource.TestCheckResourceAttr("grafana_assistant_skill.test", "body", "Terraform acceptance test skill body."),
					resource.TestCheckResourceAttr("grafana_assistant_skill.test", "include_in_knowledgebase", "true"),
					testutils.CheckLister("grafana_assistant_skill.test"),
				),
			},
			{
				ResourceName:      "grafana_assistant_skill.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testutils.TestAccExampleWithReplace(t, "resources/grafana_assistant_skill/_acc_basic.tf", map[string]string{
					"tf-acc-test-skill": "tf-acc-test-skill-updated",
					"tf-acc-command":    "tf-acc-command-updated",
				}),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("grafana_assistant_skill.test", "name", "tf-acc-test-skill-updated"),
					resource.TestCheckResourceAttr("grafana_assistant_skill.test", "command_name", "tf-acc-command-updated"),
				),
			},
		},
	})
}
