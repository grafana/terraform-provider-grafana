package provider

import (
	"fmt"
	"testing"

	onCallAPI "github.com/grafana/amixr-api-go-client"
	"github.com/grafana/terraform-provider-grafana/provider/common"
	"github.com/grafana/terraform-provider-grafana/provider/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccOnCallEscalation_basic(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	riName := fmt.Sprintf("test-acc-%s", acctest.RandString(8))
	reType := "wait"
	reDuration := 300

	resource.ParallelTest(t, resource.TestCase{
		ProviderFactories: testutils.ProviderFactories,
		CheckDestroy:      testAccCheckOnCallEscalationResourceDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccOnCallEscalationConfig(riName, reType, reDuration),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckOnCallEscalationResourceExists("grafana_oncall_escalation.test-acc-escalation"),
					resource.TestCheckResourceAttr("grafana_oncall_escalation.test-acc-escalation", "type", "wait"),
					resource.TestCheckResourceAttr("grafana_oncall_escalation.test-acc-escalation", "position", "0"),

					testAccCheckOnCallEscalationResourceExists("grafana_oncall_escalation.test-acc-escalation-repeat"),
					resource.TestCheckResourceAttr("grafana_oncall_escalation.test-acc-escalation-repeat", "type", "repeat_escalation"),
					resource.TestCheckResourceAttr("grafana_oncall_escalation.test-acc-escalation-repeat", "position", "1"),
				),
			},
		},
	})
}

func testAccCheckOnCallEscalationResourceDestroy(s *terraform.State) error {
	client := testutils.Provider.Meta().(*common.Client).OnCallClient
	for _, r := range s.RootModule().Resources {
		if r.Type != "grafana_oncall_escalation" {
			continue
		}

		if _, _, err := client.Escalations.GetEscalation(r.Primary.ID, &onCallAPI.GetEscalationOptions{}); err == nil {
			return fmt.Errorf("Escalation still exists")
		}
	}
	return nil
}

func testAccOnCallEscalationConfig(riName string, reType string, reDuration int) string {
	return fmt.Sprintf(`
resource "grafana_oncall_integration" "test-acc-integration" {
	name = "%s"
	type = "grafana"
	default_route {
	}
}

resource "grafana_oncall_escalation_chain" "test-acc-escalation-chain"{
	name = "acc-test"
}

resource "grafana_oncall_escalation" "test-acc-escalation" {
	escalation_chain_id = grafana_oncall_escalation_chain.test-acc-escalation-chain.id
	type = "%s"
	duration = "%d"
	position = 0
}

resource "grafana_oncall_escalation" "test-acc-escalation-repeat" {
	escalation_chain_id = grafana_oncall_escalation_chain.test-acc-escalation-chain.id
	type = "repeat_escalation"
	position = 1
}
`, riName, reType, reDuration)
}

func testAccCheckOnCallEscalationResourceExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("No Escalation ID is set")
		}

		client := testutils.Provider.Meta().(*common.Client).OnCallClient

		found, _, err := client.Escalations.GetEscalation(rs.Primary.ID, &onCallAPI.GetEscalationOptions{})
		if err != nil {
			return err
		}
		if found.ID != rs.Primary.ID {
			return fmt.Errorf("Escalation policy not found: %v - %v", rs.Primary.ID, found)
		}
		return nil
	}
}
