package grafana_test

import (
	"encoding/json"
	"fmt"
	"regexp"
	"testing"

	"github.com/grafana/grafana-openapi-client-go/models"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	"github.com/grafana/terraform-provider-grafana/v3/internal/testutils"
)

func TestAccAlertRule_basic(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t, ">=9.1.0")

	var group models.AlertRuleGroup

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		// Implicitly tests deletion.
		CheckDestroy: alertingRuleGroupCheckExists.destroyed(&group, nil),
		Steps: []resource.TestStep{
			// Test creation.
			{
				Config: testutils.TestAccExample(t, "resources/grafana_rule_group/resource.tf"),
				Check: resource.ComposeTestCheckFunc(
					alertingRuleGroupCheckExists.exists("grafana_rule_group.my_alert_rule", &group),
					resource.TestCheckResourceAttr("grafana_rule_group.my_alert_rule", "name", "My Rule Group"),
					resource.TestCheckResourceAttr("grafana_rule_group.my_alert_rule", "interval_seconds", "240"),
					resource.TestCheckResourceAttr("grafana_rule_group.my_alert_rule", "org_id", "1"),
					resource.TestCheckResourceAttr("grafana_rule_group.my_alert_rule", "rule.#", "1"),
					resource.TestCheckResourceAttr("grafana_rule_group.my_alert_rule", "rule.0.data.0.model", "{\"hide\":false,\"refId\":\"A\"}"),
					testutils.CheckLister("grafana_rule_group.my_alert_rule"),
				),
			},
			// Test "for: 0s"
			{
				Config: testutils.TestAccExampleWithReplace(t, "resources/grafana_rule_group/resource.tf", map[string]string{
					"2m": "0s",
				}),
				Check: resource.ComposeTestCheckFunc(
					alertingRuleGroupCheckExists.exists("grafana_rule_group.my_alert_rule", &group),
					resource.TestCheckResourceAttr("grafana_rule_group.my_alert_rule", "name", "My Rule Group"),
					resource.TestCheckResourceAttr("grafana_rule_group.my_alert_rule", "rule.#", "1"),
					resource.TestCheckResourceAttr("grafana_rule_group.my_alert_rule", "rule.0.for", "0s"),
				),
			},
			// Test import.
			{
				ResourceName:      "grafana_rule_group.my_alert_rule",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Test import without org ID.
			{
				ResourceName:      "grafana_rule_group.my_alert_rule",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs := s.RootModule().Resources["grafana_rule_group.my_alert_rule"]
					if rs == nil {
						return "", fmt.Errorf("resource not found")
					}
					return fmt.Sprintf("%s:%s", rs.Primary.Attributes["folder_uid"], rs.Primary.Attributes["name"]), nil
				},
			},
			// Test update content.
			{
				Config: testutils.TestAccExampleWithReplace(t, "resources/grafana_rule_group/resource.tf", map[string]string{
					"My Alert Rule 1": "A Different Rule",
				}),
				Check: resource.ComposeTestCheckFunc(
					alertingRuleGroupCheckExists.exists("grafana_rule_group.my_alert_rule", &group),
					resource.TestCheckResourceAttr("grafana_rule_group.my_alert_rule", "name", "My Rule Group"),
					resource.TestCheckResourceAttr("grafana_rule_group.my_alert_rule", "interval_seconds", "240"),
					resource.TestCheckResourceAttr("grafana_rule_group.my_alert_rule", "rule.#", "1"),
					resource.TestCheckResourceAttr("grafana_rule_group.my_alert_rule", "rule.0.name", "A Different Rule"),
					resource.TestCheckResourceAttr("grafana_rule_group.my_alert_rule", "rule.0.for", "2m0s"),
					resource.TestCheckResourceAttr("grafana_rule_group.my_alert_rule", "rule.0.data.0.model", "{\"hide\":false,\"refId\":\"A\"}"),
				),
			},
			// Test rename group.
			{
				Config: testutils.TestAccExampleWithReplace(t, "resources/grafana_rule_group/resource.tf", map[string]string{
					"My Rule Group": "A Different Rule Group",
				}),
				Check: resource.ComposeTestCheckFunc(
					alertingRuleGroupCheckExists.exists("grafana_rule_group.my_alert_rule", &group),
					resource.TestCheckResourceAttr("grafana_rule_group.my_alert_rule", "name", "A Different Rule Group"),
					resource.TestCheckResourceAttr("grafana_rule_group.my_alert_rule", "interval_seconds", "240"),
					resource.TestCheckResourceAttr("grafana_rule_group.my_alert_rule", "rule.#", "1"),
					resource.TestCheckResourceAttr("grafana_rule_group.my_alert_rule", "rule.0.name", "My Alert Rule 1"),
					resource.TestCheckResourceAttr("grafana_rule_group.my_alert_rule", "rule.0.for", "2m0s"),
					resource.TestCheckResourceAttr("grafana_rule_group.my_alert_rule", "rule.0.data.0.model", "{\"hide\":false,\"refId\":\"A\"}"),
				),
			},
			// Test change interval.
			{
				Config: testutils.TestAccExampleWithReplace(t, "resources/grafana_rule_group/resource.tf", map[string]string{
					"240": "360",
				}),
				Check: resource.ComposeTestCheckFunc(
					alertingRuleGroupCheckExists.exists("grafana_rule_group.my_alert_rule", &group),
					resource.TestCheckResourceAttr("grafana_rule_group.my_alert_rule", "name", "My Rule Group"),
					resource.TestCheckResourceAttr("grafana_rule_group.my_alert_rule", "interval_seconds", "360"),
					resource.TestCheckResourceAttr("grafana_rule_group.my_alert_rule", "rule.#", "1"),
				),
			},
			// Test re-parent folder.
			{
				Config: testutils.TestAccExample(t, "resources/grafana_rule_group/_acc_reparent_folder.tf"),
				Check: resource.ComposeTestCheckFunc(
					alertingRuleGroupCheckExists.exists("grafana_rule_group.my_alert_rule", &group),
					resource.TestCheckResourceAttr("grafana_rule_group.my_alert_rule", "name", "My Rule Group"),
					resource.TestCheckResourceAttr("grafana_rule_group.my_alert_rule", "interval_seconds", "240"),
					resource.TestCheckResourceAttr("grafana_rule_group.my_alert_rule", "folder_uid", "test-uid"),
					resource.TestCheckResourceAttr("grafana_rule_group.my_alert_rule", "rule.#", "1"),
					resource.TestCheckResourceAttr("grafana_rule_group.my_alert_rule", "rule.0.name", "My Alert Rule 1"),
					resource.TestCheckResourceAttr("grafana_rule_group.my_alert_rule", "rule.0.for", "2m0s"),
					resource.TestCheckResourceAttr("grafana_rule_group.my_alert_rule", "rule.0.data.0.model", "{\"hide\":false,\"refId\":\"A\"}"),
				),
			},
		},
	})
}

func TestAccAlertRule_model(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t, ">=9.1.0")

	var group models.AlertRuleGroup

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		// Implicitly tests deletion.
		CheckDestroy: alertingRuleGroupCheckExists.destroyed(&group, nil),
		Steps: []resource.TestStep{
			// Test creation.
			{
				Config: testutils.TestAccExample(t, "resources/grafana_rule_group/_acc_model_normalization.tf"),
				Check: resource.ComposeTestCheckFunc(
					// Model normalization means that default values for fields in the model JSON are not
					// included in the state, to prevent permadiffs, but non-default values must be included.
					resource.TestCheckResourceAttr("grafana_rule_group.rg_model_params_defaults",
						"rule.0.data.0.model", "{\"hide\":false,\"refId\":\"A\"}"),
					resource.TestCheckResourceAttr("grafana_rule_group.rg_model_params_omitted",
						"rule.0.data.0.model", "{\"hide\":false,\"refId\":\"A\"}"),
					resource.TestCheckResourceAttr("grafana_rule_group.rg_model_params_non_default",
						"rule.0.data.0.model", "{\"hide\":false,\"intervalMs\":1001,\"maxDataPoints\":43201,\"refId\":\"A\"}"),
				),
			},
			// Test import.
			{
				ResourceName:      "grafana_rule_group.rg_model_params_defaults",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				ResourceName:      "grafana_rule_group.rg_model_params_omitted",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				ResourceName:      "grafana_rule_group.rg_model_params_non_default",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccAlertRule_compound(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t, ">=9.1.0")

	var group models.AlertRuleGroup

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		// Implicitly tests deletion.
		CheckDestroy: alertingRuleGroupCheckExists.destroyed(&group, nil),
		Steps: []resource.TestStep{
			// Test creation.
			{
				Config: testutils.TestAccExample(t, "resources/grafana_rule_group/_acc_multi_rule_group.tf"),
				Check: resource.ComposeTestCheckFunc(
					alertingRuleGroupCheckExists.exists("grafana_rule_group.my_multi_alert_group", &group),
					resource.TestCheckResourceAttr("grafana_rule_group.my_multi_alert_group", "name", "My Multi-Alert Rule Group"),
					resource.TestCheckResourceAttr("grafana_rule_group.my_multi_alert_group", "interval_seconds", "240"),
					resource.TestCheckResourceAttr("grafana_rule_group.my_multi_alert_group", "org_id", "1"),
					resource.TestCheckResourceAttr("grafana_rule_group.my_multi_alert_group", "rule.#", "2"),
				),
			},
			// Test import.
			{
				ResourceName:      "grafana_rule_group.my_multi_alert_group",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Test update.
			{
				Config: testutils.TestAccExampleWithReplace(t, "resources/grafana_rule_group/_acc_multi_rule_group.tf", map[string]string{
					"Rule 1": "asdf",
				}),
				Check: resource.ComposeTestCheckFunc(
					alertingRuleGroupCheckExists.exists("grafana_rule_group.my_multi_alert_group", &group),
					resource.TestCheckResourceAttr("grafana_rule_group.my_multi_alert_group", "name", "My Multi-Alert Rule Group"),
					resource.TestCheckResourceAttr("grafana_rule_group.my_multi_alert_group", "rule.#", "2"),
					resource.TestCheckResourceAttr("grafana_rule_group.my_multi_alert_group", "rule.0.name", "My Alert asdf"),
					resource.TestCheckResourceAttr("grafana_rule_group.my_multi_alert_group", "rule.1.name", "My Alert Rule 2"),
				),
			},
			// Test addition of a rule to an existing group.
			{
				Config: testutils.TestAccExample(t, "resources/grafana_rule_group/_acc_multi_rule_group_added.tf"),
				Check: resource.ComposeTestCheckFunc(
					alertingRuleGroupCheckExists.exists("grafana_rule_group.my_multi_alert_group", &group),
					resource.TestCheckResourceAttr("grafana_rule_group.my_multi_alert_group", "name", "My Multi-Alert Rule Group"),
					resource.TestCheckResourceAttr("grafana_rule_group.my_multi_alert_group", "rule.#", "3"),
					resource.TestCheckResourceAttr("grafana_rule_group.my_multi_alert_group", "rule.0.name", "My Alert Rule 1"),
					resource.TestCheckResourceAttr("grafana_rule_group.my_multi_alert_group", "rule.1.name", "My Alert Rule 2"),
					resource.TestCheckResourceAttr("grafana_rule_group.my_multi_alert_group", "rule.2.name", "My Alert Rule 3"),
				),
			},
			// Test removal of rules from an existing group.
			{
				Config: testutils.TestAccExample(t, "resources/grafana_rule_group/_acc_multi_rule_group_subtracted.tf"),
				Check: resource.ComposeTestCheckFunc(
					alertingRuleGroupCheckExists.exists("grafana_rule_group.my_multi_alert_group", &group),
					resource.TestCheckResourceAttr("grafana_rule_group.my_multi_alert_group", "name", "My Multi-Alert Rule Group"),
					resource.TestCheckResourceAttr("grafana_rule_group.my_multi_alert_group", "rule.#", "2"),
					resource.TestCheckResourceAttr("grafana_rule_group.my_multi_alert_group", "rule.0.name", "My Alert Rule 1"),
					resource.TestCheckResourceAttr("grafana_rule_group.my_multi_alert_group", "rule.1.name", "My Alert Rule 2"),
				),
			},
		},
	})
}

func TestAccAlertRule_inOrg(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t, ">=9.1.0")

	var group models.AlertRuleGroup
	var org models.OrgDetailsDTO
	name := "test:" + acctest.RandString(10)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             orgCheckExists.destroyed(&org, nil),
		Steps: []resource.TestStep{
			// Test creation.
			{
				Config: testAccAlertRuleGroupInOrgConfig(name, 240, false),
				Check: resource.ComposeTestCheckFunc(
					alertingRuleGroupCheckExists.exists("grafana_rule_group.test", &group),
					orgCheckExists.exists("grafana_organization.test", &org),
					checkResourceIsInOrg("grafana_rule_group.test", "grafana_organization.test"),
					resource.TestCheckResourceAttr("grafana_rule_group.test", "name", name),
					resource.TestCheckResourceAttr("grafana_rule_group.test", "interval_seconds", "240"),
					resource.TestCheckResourceAttr("grafana_rule_group.test", "rule.#", "1"),
					resource.TestCheckResourceAttr("grafana_rule_group.test", "rule.0.data.0.model", "{\"hide\":false,\"refId\":\"A\"}"),
				),
			},
			// Test update content.
			{
				Config: testAccAlertRuleGroupInOrgConfig(name, 360, false),
				Check: resource.ComposeTestCheckFunc(
					alertingRuleGroupCheckExists.exists("grafana_rule_group.test", &group),
					orgCheckExists.exists("grafana_organization.test", &org),
					checkResourceIsInOrg("grafana_rule_group.test", "grafana_organization.test"),
					resource.TestCheckResourceAttr("grafana_rule_group.test", "name", name),
					resource.TestCheckResourceAttr("grafana_rule_group.test", "interval_seconds", "360"),
					resource.TestCheckResourceAttr("grafana_rule_group.test", "rule.#", "1"),
					resource.TestCheckResourceAttr("grafana_rule_group.test", "rule.0.data.0.model", "{\"hide\":false,\"refId\":\"A\"}"),
				),
			},
			// Test import.
			{
				ResourceName:      "grafana_rule_group.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Test delete resource, but not org.
			{
				Config: testutils.WithoutResource(t, testAccAlertRuleGroupInOrgConfig(name, 360, false), "grafana_rule_group.test"),
				Check: resource.ComposeTestCheckFunc(
					orgCheckExists.exists("grafana_organization.test", &org),
					alertingRuleGroupCheckExists.destroyed(&group, &org),
				),
			},
		},
	})
}

func TestAccAlertRule_nameConflict(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t, ">=9.1.0")

	name := acctest.RandString(10)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "grafana_organization" "test" {
	name = "%[1]s"
}

resource "grafana_folder" "test" {
	org_id = grafana_organization.test.id
	title = "%[1]s-test"
}

resource "grafana_rule_group" "first" {
	org_id = grafana_organization.test.id
	name             = "%[1]s"
	folder_uid       = grafana_folder.test.uid
	interval_seconds = 60
	rule {
		name           = "My Alert Rule first"
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

resource "grafana_rule_group" "second" {
	depends_on = [ grafana_rule_group.first ]
	org_id = grafana_organization.test.id
	name             = "%[1]s"
	folder_uid       = grafana_folder.test.uid
	interval_seconds = 60
	rule {
		name           = "My Alert Rule second"
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
				`, name),
				ExpectError: regexp.MustCompile(`rule group with name "` + name + `" already exists`),
			},
		},
	})
}

func TestAccAlertRule_ruleNameConflict(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t, ">=9.1.0")

	name := acctest.RandString(10)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "grafana_organization" "test" {
	name = "%[1]s"
}

resource "grafana_folder" "first" {
	org_id = grafana_organization.test.id
	title = "%[1]s-first"
}

resource "grafana_rule_group" "first" {
	org_id = grafana_organization.test.id
	name             = "alert rule group"
	folder_uid       = grafana_folder.first.uid
	interval_seconds = 60
	rule {
		name           = "My Alert Rule"
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
	rule {
		name           = "My Alert Rule"
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
				`, name),
				ExpectError: regexp.MustCompile(`rule with name "My Alert Rule" is defined more than once`),
			},
		},
	})
}

func TestAccAlertRule_ruleUIDConflict(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t, ">=9.1.0")

	name := acctest.RandString(10)
	uid := acctest.RandString(10)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "grafana_organization" "test" {
	name = "%[1]s"
}

resource "grafana_folder" "first" {
	org_id = grafana_organization.test.id
	title = "%[1]s-first"
}

resource "grafana_rule_group" "first" {
	org_id = grafana_organization.test.id
	name             = "alert rule group"
	folder_uid       = grafana_folder.first.uid
	interval_seconds = 60
	rule {
		name           = "%[1]s"
		uid            = "%[2]s"
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
	rule {
		name           = "%[1]s 2"
		uid            = "%[2]s"
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
				`, name, uid),
				ExpectError: regexp.MustCompile(`rule with UID "` + uid + `" is defined more than once. Rules with name "` + name + `" and "` + name + ` 2" have the same uid`),
			},
		},
	})
}

func TestAccAlertRule_moveRules(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t, ">=9.1.0")

	name := acctest.RandString(10)
	ruleFunc := func(ruleName string) string {
		return fmt.Sprintf(`
	rule {
		name           = "%[1]s"
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
	`, ruleName)
	}

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "grafana_folder" "test" {
	title = "%[1]s"
	uid   = "%[1]s"
}

resource "grafana_rule_group" "first" {
	name             = "%[1]s-first"
	folder_uid       = grafana_folder.test.uid
	interval_seconds = 60
	%[2]s
	%[3]s
}

resource "grafana_rule_group" "second" {
	name             = "%[1]s-second"
	folder_uid       = grafana_folder.test.uid
	interval_seconds = 60
	%[4]s
}
				`, name, ruleFunc("My Alert Rule 1"), ruleFunc("My Alert Rule 2"), ruleFunc("My Alert Rule 3")),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("grafana_rule_group.first", "rule.#", "2"),
					resource.TestCheckResourceAttr("grafana_rule_group.first", "rule.0.name", "My Alert Rule 1"),
					resource.TestCheckResourceAttr("grafana_rule_group.first", "rule.1.name", "My Alert Rule 2"),
					resource.TestCheckResourceAttr("grafana_rule_group.second", "rule.#", "1"),
					resource.TestCheckResourceAttr("grafana_rule_group.second", "rule.0.name", "My Alert Rule 3"),
				),
			},
			{
				Config: fmt.Sprintf(`
resource "grafana_folder" "test" {
	title = "%[1]s"
	uid   = "%[1]s"
}

resource "grafana_rule_group" "first" {
	name             = "%[1]s-first"
	folder_uid       = grafana_folder.test.uid
	interval_seconds = 60
	%[2]s
}

resource "grafana_rule_group" "second" {
	name             = "%[1]s-second"
	folder_uid       = grafana_folder.test.uid
	interval_seconds = 60
	%[3]s
	%[4]s
}`, name, ruleFunc("My Alert Rule 1"), ruleFunc("My Alert Rule 2"), ruleFunc("My Alert Rule 3")),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("grafana_rule_group.first", "rule.#", "1"),
					resource.TestCheckResourceAttr("grafana_rule_group.first", "rule.0.name", "My Alert Rule 1"),
					resource.TestCheckResourceAttr("grafana_rule_group.second", "rule.#", "2"),
					resource.TestCheckResourceAttr("grafana_rule_group.second", "rule.0.name", "My Alert Rule 2"),
					resource.TestCheckResourceAttr("grafana_rule_group.second", "rule.1.name", "My Alert Rule 3"),
				),
			},
		},
	})
}

func TestAccAlertRule_disableProvenance(t *testing.T) {
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
				Config: testAccAlertRuleGroupInOrgConfig(name, 240, false),
				Check: resource.ComposeTestCheckFunc(
					alertingRuleGroupCheckExists.exists("grafana_rule_group.test", &group),
					orgCheckExists.exists("grafana_organization.test", &org),
					checkResourceIsInOrg("grafana_rule_group.test", "grafana_organization.test"),
					resource.TestCheckResourceAttr("grafana_rule_group.test", "name", name),
					resource.TestCheckResourceAttr("grafana_rule_group.test", "disable_provenance", "false"),
				),
			},
			// Test import.
			{
				ResourceName:      "grafana_rule_group.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Enable editing from UI.
			{
				Config: testAccAlertRuleGroupInOrgConfig(name, 240, true),
				Check: resource.ComposeTestCheckFunc(
					alertingRuleGroupCheckExists.exists("grafana_rule_group.test", &group),
					orgCheckExists.exists("grafana_organization.test", &org),
					checkResourceIsInOrg("grafana_rule_group.test", "grafana_organization.test"),
					resource.TestCheckResourceAttr("grafana_rule_group.test", "name", name),
					resource.TestCheckResourceAttr("grafana_rule_group.test", "disable_provenance", "true"),
				),
			},
			// Test import.
			{
				ResourceName:      "grafana_rule_group.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Disable editing from UI.
			{
				Config: testAccAlertRuleGroupInOrgConfig(name, 240, false),
				Check: resource.ComposeTestCheckFunc(
					alertingRuleGroupCheckExists.exists("grafana_rule_group.test", &group),
					orgCheckExists.exists("grafana_organization.test", &org),
					checkResourceIsInOrg("grafana_rule_group.test", "grafana_organization.test"),
					resource.TestCheckResourceAttr("grafana_rule_group.test", "name", name),
					resource.TestCheckResourceAttr("grafana_rule_group.test", "disable_provenance", "false"),
				),
			},
		},
	})
}

func TestAccAlertRule_zeroSeconds(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t, ">=9.1.0")

	var group models.AlertRuleGroup
	var name = acctest.RandString(10)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             alertingRuleGroupCheckExists.destroyed(&group, nil),
		Steps: []resource.TestStep{
			{
				Config: testAccAlertRuleZeroSeconds(name),
				Check: resource.ComposeTestCheckFunc(
					alertingRuleGroupCheckExists.exists("grafana_rule_group.my_rule_group", &group),
					resource.TestCheckResourceAttr("grafana_rule_group.my_rule_group", "name", name),
					resource.TestCheckResourceAttr("grafana_rule_group.my_rule_group", "rule.#", "1"),
					resource.TestCheckResourceAttr("grafana_rule_group.my_rule_group", "rule.0.name", "My Random Walk Alert"),
					resource.TestCheckResourceAttr("grafana_rule_group.my_rule_group", "rule.0.for", "0s"),
				),
			},
		},
	})
}

func TestAccAlertRule_NotificationSettings(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t, ">=10.4.0")

	var group models.AlertRuleGroup
	var name = acctest.RandString(10)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             alertingRuleGroupCheckExists.destroyed(&group, nil),
		Steps: []resource.TestStep{
			{
				Config: testAccAlertRuleWithNotificationSettings(name, []string{"alertname", "grafana_folder", "test"}),
				Check: resource.ComposeTestCheckFunc(
					alertingRuleGroupCheckExists.exists("grafana_rule_group.my_rule_group", &group),
					resource.TestCheckResourceAttr("grafana_rule_group.my_rule_group", "name", name),
					resource.TestCheckResourceAttr("grafana_rule_group.my_rule_group", "rule.#", "1"),
					resource.TestCheckResourceAttr("grafana_rule_group.my_rule_group", "rule.0.name", fmt.Sprintf("%s-alertrule", name)),
					resource.TestCheckResourceAttr("grafana_rule_group.my_rule_group", "rule.0.notification_settings.0.contact_point", fmt.Sprintf("%s-receiver", name)),
					resource.TestCheckResourceAttr("grafana_rule_group.my_rule_group", "rule.0.notification_settings.0.group_wait", "45s"),
					resource.TestCheckResourceAttr("grafana_rule_group.my_rule_group", "rule.0.notification_settings.0.group_interval", "6m"),
					resource.TestCheckResourceAttr("grafana_rule_group.my_rule_group", "rule.0.notification_settings.0.repeat_interval", "3h"),
					resource.TestCheckResourceAttr("grafana_rule_group.my_rule_group", "rule.0.notification_settings.0.mute_timings.#", "1"),
					resource.TestCheckResourceAttr("grafana_rule_group.my_rule_group", "rule.0.notification_settings.0.mute_timings.0", fmt.Sprintf("%s-mute-timing", name)),
					resource.TestCheckResourceAttr("grafana_rule_group.my_rule_group", "rule.0.notification_settings.0.group_by.#", "3"),
					resource.TestCheckResourceAttr("grafana_rule_group.my_rule_group", "rule.0.notification_settings.0.group_by.0", "alertname"),
					resource.TestCheckResourceAttr("grafana_rule_group.my_rule_group", "rule.0.notification_settings.0.group_by.1", "grafana_folder"),
					resource.TestCheckResourceAttr("grafana_rule_group.my_rule_group", "rule.0.notification_settings.0.group_by.2", "test"),
				),
			},
		},
	})
}

func TestAccRecordingRule(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t) // TODO: change to 11.3.1 when available

	var group models.AlertRuleGroup
	var name = acctest.RandString(10)
	var metric = "valid_metric"

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             alertingRuleGroupCheckExists.destroyed(&group, nil),
		Steps: []resource.TestStep{
			{
				Config: testAccRecordingRule(name, metric, "A"),
				Check: resource.ComposeTestCheckFunc(
					alertingRuleGroupCheckExists.exists("grafana_rule_group.my_rule_group", &group),
					resource.TestCheckResourceAttr("grafana_rule_group.my_rule_group", "name", name),
					resource.TestCheckResourceAttr("grafana_rule_group.my_rule_group", "rule.#", "1"),
					resource.TestCheckResourceAttr("grafana_rule_group.my_rule_group", "rule.0.name", "My Random Walk Alert"),
					resource.TestCheckResourceAttr("grafana_rule_group.my_rule_group", "rule.0.data.0.model", "{\"refId\":\"A\"}"),
					resource.TestCheckResourceAttr("grafana_rule_group.my_rule_group", "rule.0.record.0.metric", metric),
					resource.TestCheckResourceAttr("grafana_rule_group.my_rule_group", "rule.0.record.0.from", "A"),
					// ensure fields are cleared as expected
					resource.TestCheckResourceAttr("grafana_rule_group.my_rule_group", "rule.0.for", "2m0s"),
					resource.TestCheckResourceAttr("grafana_rule_group.my_rule_group", "rule.0.condition", "A"),
					resource.TestCheckResourceAttr("grafana_rule_group.my_rule_group", "rule.0.no_data_state", "NoData"),
					resource.TestCheckResourceAttr("grafana_rule_group.my_rule_group", "rule.0.exec_err_state", "Alerting"),
				),
			},
		},
	})
}

func testAccAlertRuleGroupInOrgConfig(name string, interval int, disableProvenance bool) string {
	return fmt.Sprintf(`
resource "grafana_organization" "test" {
	name = "%[1]s"
}

resource "grafana_folder" "test" {
	org_id = grafana_organization.test.id
	title = "%[1]s"
}

resource "grafana_rule_group" "test" {
	org_id = grafana_organization.test.id
	name             = "%[1]s"
	folder_uid       = grafana_folder.test.uid
	interval_seconds = %[2]d
	disable_provenance = %[3]t
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
`, name, interval, disableProvenance)
}

func testAccAlertRuleZeroSeconds(name string) string {
	return fmt.Sprintf(`
resource "grafana_folder" "rule_folder" {
	title = "%[1]s"
}

resource "grafana_data_source" "testdata_datasource" {
	name = "%[1]s"
	type = "grafana-testdata-datasource"
	url  = "http://localhost:3333"
}

resource "grafana_rule_group" "my_rule_group" {
	name             = "%[1]s"
	folder_uid       = grafana_folder.rule_folder.uid
	interval_seconds = 60
	org_id           = 1

	rule {
		name      = "My Random Walk Alert"
		condition = "C"
		for       = "0s"

		// Query the datasource.
		data {
			ref_id = "A"
			relative_time_range {
				from = 600
				to   = 0
			}
			datasource_uid = grafana_data_source.testdata_datasource.uid
			model = jsonencode({
				intervalMs    = 1000
				maxDataPoints = 43200
				refId         = "A"
			})
		}
	}
}`, name)
}

func testAccAlertRuleWithNotificationSettings(name string, groupBy []string) string {
	gr := ""
	if len(groupBy) > 0 {
		b, _ := json.Marshal(groupBy)
		gr = "group_by = " + string(b)
	}
	return fmt.Sprintf(`
resource "grafana_folder" "rule_folder" {
	title = "%[1]s"
}

resource "grafana_data_source" "testdata_datasource" {
	name = "%[1]s"
	type = "grafana-testdata-datasource"
	url  = "http://localhost:3333"
}

resource "grafana_mute_timing" "my_mute_timing" {
		name = "%[1]s-mute-timing"
		intervals {}
}

resource "grafana_contact_point" "my_contact_point" {
	name      = "%[1]s-receiver"
	email {
		addresses = [ "hello@example.com" ]
	}
}

resource "grafana_rule_group" "my_rule_group" {
	name             = "%[1]s"
	folder_uid       = grafana_folder.rule_folder.uid
	interval_seconds = 60

	rule {
		name      = "%[1]s-alertrule"
		condition = "C"
		for       = "0s"

		// Query the datasource.
		data {
			ref_id = "A"
			relative_time_range {
				from = 600
				to   = 0
			}
			datasource_uid = grafana_data_source.testdata_datasource.uid
			model = jsonencode({
				intervalMs    = 1000
				maxDataPoints = 43200
				refId         = "A"
			})
		}

		notification_settings {
			contact_point = grafana_contact_point.my_contact_point.name
			%[2]s
			group_wait      = "45s"
            group_interval  = "6m"
            repeat_interval = "3h"
			mute_timings = [grafana_mute_timing.my_mute_timing.name]
		}
	}
}`, name, gr)
}

func testAccRecordingRule(name string, metric string, refID string) string {
	return fmt.Sprintf(`
resource "grafana_folder" "rule_folder" {
	title = "%[1]s"
}

resource "grafana_data_source" "testdata_datasource" {
	name = "%[1]s"
	type = "grafana-testdata-datasource"
	url  = "http://localhost:3333"
}

resource "grafana_rule_group" "my_rule_group" {
	name             = "%[1]s"
	folder_uid       = grafana_folder.rule_folder.uid
	interval_seconds = 60

	rule {
		name      = "My Random Walk Alert"
		// following should be cleared by Grafana
		condition = "A"
		no_data_state  = "NoData"
		exec_err_state = "Alerting"
		for = "2m"

		// Query the datasource.
		data {
			ref_id = "A"
			relative_time_range {
				from = 600
				to   = 0
			}
			datasource_uid = grafana_data_source.testdata_datasource.uid
			model = jsonencode({
				intervalMs    = 1000
				maxDataPoints = 43200
				refId         = "A"
			})
		}
		record {
			metric = "%[2]s"
			from   = "%[3]s"
		}
	}
}`, name, metric, refID)
}
