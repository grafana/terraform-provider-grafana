package oncall_test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceEscalationPolicy_Basic(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	randomName := fmt.Sprintf("test-acc-%s", acctest.RandString(8))

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceEscalationPolicyConfig(randomName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.grafana_oncall_escalation_policy.test", "id"),
					resource.TestCheckResourceAttrPair(
						"grafana_oncall_escalation.test", "id",
						"data.grafana_oncall_escalation_policy.test", "id",
					),
					resource.TestCheckResourceAttrPair(
						"grafana_oncall_escalation.test", "type",
						"data.grafana_oncall_escalation_policy.test", "type",
					),
				),
			},
		},
	})
}

func TestAccDataSourceEscalationPolicy_NotFound(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccDataSourceEscalationPolicyNotFoundConfig(),
				ExpectError: regexp.MustCompile(`couldn't find an escalation policy matching`),
			},
		},
	})
}

func testAccDataSourceEscalationPolicyNotFoundConfig() string {
	return `
data "grafana_oncall_escalation_policy" "test" {
	escalation_chain_id = "nonexistent"
	position            = 999
}
`
}

func testAccDataSourceEscalationPolicyConfig(name string) string {
	return fmt.Sprintf(`
resource "grafana_oncall_escalation_chain" "test" {
	name = "%[1]s"
}

resource "grafana_oncall_escalation" "test" {
	escalation_chain_id = grafana_oncall_escalation_chain.test.id
	type                = "wait"
	duration            = 300
	position            = 0
}

data "grafana_oncall_escalation_policy" "test" {
	escalation_chain_id = grafana_oncall_escalation_chain.test.id
	position            = grafana_oncall_escalation.test.position
}
`, name)
}
