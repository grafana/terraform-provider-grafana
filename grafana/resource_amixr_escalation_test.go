package grafana

import (
	"fmt"
	"testing"

	amixrAPI "github.com/grafana/amixr-api-go-client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccAmixrEscalation_basic(t *testing.T) {
	CheckCloudInstanceTestsEnabled(t)

	riName := fmt.Sprintf("test-acc-%s", acctest.RandString(8))
	reType := "wait"
	reDuration := 300

	resource.Test(t, resource.TestCase{
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckAmixrEscalationResourceDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAmixrEscalationConfig(riName, reType, reDuration),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAmixrEscalationResourceExists("amixr_escalation.test-acc-escalation"),
					resource.TestCheckResourceAttr(
						"amixr_escalation.test-acc-escalation", "type", "wait",
					),
				),
			},
		},
	})
}

func testAccCheckAmixrEscalationResourceDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*amixrAPI.Client)
	for _, r := range s.RootModule().Resources {
		if r.Type != "amixr_escalation" {
			continue
		}

		if _, _, err := client.Escalations.GetEscalation(r.Primary.ID, &amixrAPI.GetEscalationOptions{}); err == nil {
			return fmt.Errorf("Escalation still exists")
		}
	}
	return nil
}

func testAccAmixrEscalationConfig(riName string, reType string, reDuration int) string {
	return fmt.Sprintf(`
resource "amixr_integration" "test-acc-integration" {
	name = "%s"
	type = "grafana"
}

resource "amixr_escalation_chain" "test-acc-escalation-chain"{
	name = "acc-test"
}

resource "amixr_escalation" "test-acc-escalation" {
	escalation_chain_id = amixr_escalation_chain.test-acc-escalation-chain.id
	type = "%s"
	duration = "%d"
	position = 0
}
`, riName, reType, reDuration)
}

func testAccCheckAmixrEscalationResourceExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("No Escalation ID is set")
		}

		client := testAccProvider.Meta().(*amixrAPI.Client)

		found, _, err := client.Escalations.GetEscalation(rs.Primary.ID, &amixrAPI.GetEscalationOptions{})
		if err != nil {
			return err
		}
		if found.ID != rs.Primary.ID {
			return fmt.Errorf("Escalation policy not found: %v - %v", rs.Primary.ID, found)
		}
		return nil
	}
}
