package grafana_test

import (
	"fmt"
	"testing"

	"github.com/grafana/grafana-openapi-client-go/models"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"

	"github.com/grafana/terraform-provider-grafana/v3/internal/testutils"
)

func TestAccRuleGroupConfig_basic(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t, ">=9.1.0")

	var group models.AlertRuleGroup
	name := acctest.RandString(10)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             alertingRuleGroupCheckExists.destroyed(&group, nil),
		Steps: []resource.TestStep{
			{
				Config: testAccRuleGroupConfigBasic(name, 240),
				Check: resource.ComposeTestCheckFunc(
					alertingRuleGroupCheckExists.exists("grafana_rule_group_config.test", &group),
					resource.TestCheckResourceAttr("grafana_rule_group_config.test", "rule_group_name", name),
					resource.TestCheckResourceAttr("grafana_rule_group_config.test", "interval_seconds", "240"),
				),
			},
			{
				ResourceName:      "grafana_rule_group_config.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccRuleGroupConfigBasic(name, 360),
				Check: resource.ComposeTestCheckFunc(
					alertingRuleGroupCheckExists.exists("grafana_rule_group_config.test", &group),
					resource.TestCheckResourceAttr("grafana_rule_group_config.test", "rule_group_name", name),
					resource.TestCheckResourceAttr("grafana_rule_group_config.test", "interval_seconds", "360"),
				),
			},
		},
	})
}

func TestAccRuleGroupConfig_inOrg(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t, ">=9.1.0")

	var group models.AlertRuleGroup
	var org models.OrgDetailsDTO
	name := acctest.RandString(10)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy: resource.ComposeTestCheckFunc(
			orgCheckExists.destroyed(&org, nil),
			alertingRuleGroupCheckExists.destroyed(&group, &org),
		),
		Steps: []resource.TestStep{
			{
				Config: testAccRuleGroupConfigInOrg(name, 240),
				Check: resource.ComposeTestCheckFunc(
					alertingRuleGroupCheckExists.exists("grafana_rule_group_config.test", &group),
					orgCheckExists.exists("grafana_organization.test", &org),
					checkResourceIsInOrg("grafana_rule_group_config.test", "grafana_organization.test"),
					resource.TestCheckResourceAttr("grafana_rule_group_config.test", "rule_group_name", name),
					resource.TestCheckResourceAttr("grafana_rule_group_config.test", "interval_seconds", "240"),
				),
			},
			{
				ResourceName:      "grafana_rule_group_config.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccRuleGroupConfigInOrg(name, 360),
				Check: resource.ComposeTestCheckFunc(
					alertingRuleGroupCheckExists.exists("grafana_rule_group_config.test", &group),
					orgCheckExists.exists("grafana_organization.test", &org),
					checkResourceIsInOrg("grafana_rule_group_config.test", "grafana_organization.test"),
					resource.TestCheckResourceAttr("grafana_rule_group_config.test", "rule_group_name", name),
					resource.TestCheckResourceAttr("grafana_rule_group_config.test", "interval_seconds", "360"),
				),
			},
		},
	})
}

func testAccRuleGroupConfigBasic(name string, interval int) string {
	return fmt.Sprintf(`
resource "grafana_folder" "test" {
	title = "%[1]s"
}

resource "grafana_rule_group" "test" {
	name             = "%[1]s"
	folder_uid       = grafana_folder.test.uid
	interval_seconds = %[2]d
	rule {
		name           = "My Alert Rule 1"
		for            = "2m"
		condition      = "B"
		no_data_state  = "NoData"
		exec_err_state = "Alerting"
		is_paused = false
		data {
			ref_id     = "A"
			query_type = ""
			relative_time_range {
				from = 600
				to   = 0
			}
			datasource_uid = "PD8C576611E62080A"
			model = jsonencode({
				hide          = false
				intervalMs    = 1000
				maxDataPoints = 43200
				refId         = "A"
			})
		}
	}
}

resource "grafana_rule_group_config" "test" {
	folder_uid      = grafana_folder.test.uid
	rule_group_name = grafana_rule_group.test.name
	interval_seconds = %[2]d
}
`, name, interval)
}

func testAccRuleGroupConfigInOrg(name string, interval int) string {
	return fmt.Sprintf(`
resource "grafana_organization" "test" {
	name = "%[1]s"
}

resource "grafana_folder" "test" {
	org_id = grafana_organization.test.id
	title = "%[1]s"
}

resource "grafana_rule_group" "test" {
	org_id          = grafana_organization.test.id
	name             = "%[1]s"
	folder_uid       = grafana_folder.test.uid
	interval_seconds = %[2]d
	rule {
		name           = "My Alert Rule 1"
		for            = "2m"
		condition      = "B"
		no_data_state  = "NoData"
		exec_err_state = "Alerting"
		is_paused = false
		data {
			ref_id     = "A"
			query_type = ""
			relative_time_range {
				from = 600
				to   = 0
			}
			datasource_uid = "PD8C576611E62080A"
			model = jsonencode({
				hide          = false
				intervalMs    = 1000
				maxDataPoints = 43200
				refId         = "A"
			})
		}
	}
}

resource "grafana_rule_group_config" "test" {
	org_id          = grafana_organization.test.id
	folder_uid      = grafana_folder.test.uid
	rule_group_name = grafana_rule_group.test.name
	interval_seconds = %[2]d
}
`, name, interval)
}