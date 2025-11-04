package oncall_test

import (
	"fmt"
	"regexp"
	"testing"

	onCallAPI "github.com/grafana/amixr-api-go-client"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
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
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckOnCallEscalationResourceDestroy,
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

					testAccCheckOnCallEscalationResourceExists("grafana_oncall_escalation.test-acc-escalation-policy-team"),
					resource.TestCheckResourceAttr("grafana_oncall_escalation.test-acc-escalation-policy-team", "type", "notify_team_members"),
					resource.TestCheckResourceAttr("grafana_oncall_escalation.test-acc-escalation-policy-team", "position", "2"),
					resource.TestCheckResourceAttrSet("grafana_oncall_escalation.test-acc-escalation-policy-team", "notify_to_team_members"),

					testAccCheckOnCallEscalationResourceExists("grafana_oncall_escalation.test-acc-escalation-policy-declare-incident"),
					resource.TestCheckResourceAttr("grafana_oncall_escalation.test-acc-escalation-policy-declare-incident", "type", "declare_incident"),
					resource.TestCheckResourceAttr("grafana_oncall_escalation.test-acc-escalation-policy-declare-incident", "position", "3"),
					resource.TestCheckResourceAttrSet("grafana_oncall_escalation.test-acc-escalation-policy-declare-incident", "severity"),

					testAccCheckOnCallEscalationResourceExists("grafana_oncall_escalation.test-acc-escalation-policy-num-alerts"),
					resource.TestCheckResourceAttr("grafana_oncall_escalation.test-acc-escalation-policy-num-alerts", "type", "notify_if_num_alerts_in_window"),
					resource.TestCheckResourceAttr("grafana_oncall_escalation.test-acc-escalation-policy-num-alerts", "position", "4"),
					resource.TestCheckResourceAttr("grafana_oncall_escalation.test-acc-escalation-policy-num-alerts", "num_alerts_in_window", "3"),
					resource.TestCheckResourceAttr("grafana_oncall_escalation.test-acc-escalation-policy-num-alerts", "num_minutes_in_window", "5"),
				),
			},
			{
				ImportState:       true,
				ResourceName:      "grafana_oncall_escalation.test-acc-escalation",
				ImportStateVerify: true,
			},
			{
				ImportState:       true,
				ResourceName:      "grafana_oncall_escalation.test-acc-escalation-repeat",
				ImportStateVerify: true,
			},
			{
				ImportState:       true,
				ResourceName:      "grafana_oncall_escalation.test-acc-escalation-policy-team",
				ImportStateVerify: true,
			},
			{
				ImportState:       true,
				ResourceName:      "grafana_oncall_escalation.test-acc-escalation-policy-num-alerts",
				ImportStateVerify: true,
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
	name = "acc-test-%s"
}

resource "grafana_team" "test-acc-team" {
	name = "acc-escalation-test-%s"
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

data "grafana_oncall_team" "test-acc-team" {
	name = grafana_team.test-acc-team.name
}

resource "grafana_oncall_escalation" "test-acc-escalation-policy-team" {
	escalation_chain_id = grafana_oncall_escalation_chain.test-acc-escalation-chain.id
	type = "notify_team_members"
	notify_to_team_members = data.grafana_oncall_team.test-acc-team.id
	position = 2
}

resource "grafana_oncall_escalation" "test-acc-escalation-policy-declare-incident" {
	escalation_chain_id = grafana_oncall_escalation_chain.test-acc-escalation-chain.id
	type = "declare_incident"
	severity = "critical"
	position = 3
}

resource "grafana_oncall_escalation" "test-acc-escalation-policy-num-alerts" {
	escalation_chain_id = grafana_oncall_escalation_chain.test-acc-escalation-chain.id
	type = "notify_if_num_alerts_in_window"
	num_alerts_in_window = 3
	num_minutes_in_window = 5
	position = 4
}
`, riName, riName, riName, reType, reDuration)
}

func TestAccOnCallEscalation_notifyIfNumAlertsInWindow_wrongType(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	riName := fmt.Sprintf("test-acc-%s", acctest.RandString(8))

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccOnCallEscalationNotifyIfNumAlertsInWindowConfigWrongType(riName),
				ExpectError: regexp.MustCompile(`.*num_alerts_in_window can not be set with type: wait.*`),
			},
		},
	})
}

func testAccOnCallEscalationNotifyIfNumAlertsInWindowConfigWrongType(riName string) string {
	return fmt.Sprintf(`
resource "grafana_oncall_integration" "test-acc-integration" {
	name = "%s"
	type = "grafana"
	default_route {
	}
}

resource "grafana_oncall_escalation_chain" "test-acc-escalation-chain" {
	name = "acc-test-%s"
}

resource "grafana_oncall_escalation" "test-acc-escalation-num-alerts" {
	escalation_chain_id = grafana_oncall_escalation_chain.test-acc-escalation-chain.id
	type = "wait"
	duration = 300
	num_alerts_in_window = 3
	num_minutes_in_window = 5
	position = 0
}
`, riName, riName)
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
