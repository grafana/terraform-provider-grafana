package assistant_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"

	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
)

func TestAccAssistantRule_basic(t *testing.T) {
	testutils.CheckAssistantTestsEnabled(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testutils.TestAccExample(t, "resources/grafana_assistant_rule/_acc_basic.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("grafana_assistant_rule.test", "id"),
					resource.TestCheckResourceAttr("grafana_assistant_rule.test", "name", "tf-acc-test-rule"),
					resource.TestCheckResourceAttr("grafana_assistant_rule.test", "scope", "tenant"),
					resource.TestCheckResourceAttr("grafana_assistant_rule.test", "rule_content", "Terraform acceptance test rule."),
				),
			},
			{
				ResourceName:      "grafana_assistant_rule.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testutils.TestAccExampleWithReplace(t, "resources/grafana_assistant_rule/_acc_basic.tf", map[string]string{
					"tf-acc-test-rule": "tf-acc-test-rule-updated",
				}),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("grafana_assistant_rule.test", "name", "tf-acc-test-rule-updated"),
				),
			},
		},
	})
}

func TestAccAssistantMCPServer_basic(t *testing.T) {
	testutils.CheckAssistantTestsEnabled(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testutils.TestAccExample(t, "resources/grafana_assistant_mcp_server/_acc_basic.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("grafana_assistant_mcp_server.test", "id"),
					resource.TestCheckResourceAttr("grafana_assistant_mcp_server.test", "name", "tf-acc-test-mcp"),
					resource.TestCheckResourceAttr("grafana_assistant_mcp_server.test", "scope", "tenant"),
					resource.TestCheckResourceAttr("grafana_assistant_mcp_server.test", "configuration.url", "https://httpbin.org/anything"),
				),
			},
			{
				ResourceName:      "grafana_assistant_mcp_server.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"custom_headers",
				},
			},
			{
				Config: testutils.TestAccExampleWithReplace(t, "resources/grafana_assistant_mcp_server/_acc_basic.tf", map[string]string{
					"tf-acc-test-mcp": "tf-acc-test-mcp-updated",
				}),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("grafana_assistant_mcp_server.test", "name", "tf-acc-test-mcp-updated"),
				),
			},
		},
	})
}
