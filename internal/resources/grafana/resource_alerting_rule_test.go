package grafana_test

import (
	"fmt"
	"testing"

	"github.com/grafana/grafana-openapi-client-go/models"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	"github.com/grafana/terraform-provider-grafana/v3/internal/testutils"
)

func TestAccAlertRuleSeparate_compound(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t, ">=9.1.0")

	var alertRule models.ProvisionedAlertRule

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		// Implicitly tests deletion.
		CheckDestroy: alertingRuleCheckExists.destroyed(&alertRule, nil),
		Steps: []resource.TestStep{
			// Test creation.
			{
				Config: testutils.TestAccExample(t, "resources/grafana_rule/resource.tf"),
				Check: resource.ComposeTestCheckFunc(
					alertingRuleCheckExists.exists("grafana_rule.test_rule", &alertRule),
					resource.TestCheckResourceAttr("grafana_rule.test_rule", "name", "My Alert Rule"),
					testutils.CheckLister("grafana_rule.test_rule"),
				),
			},
			// Test update
			{
				Config: testutils.TestAccExampleWithReplace(t, "resources/grafana_rule/resource.tf", map[string]string{
					"My Alert Rule": "Our Alert Rule",
				}),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("grafana_rule.test_rule", "name", "Our Alert Rule"),
				),
			},
		},
	})
}

func TestAccAlertRule_createAndUpdate(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t, ">=9.1.0")

	var alertRule models.ProvisionedAlertRule
	name := acctest.RandString(10)
	folderName := acctest.RandString(10)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		// Implicitly tests deletion.
		CheckDestroy: alertingRuleCheckExists.destroyed(&alertRule, nil),
		Steps: []resource.TestStep{
			// Test creation.
			{
				Config: testAccAlertRuleBasic(name, folderName),
				Check: resource.ComposeTestCheckFunc(
					alertingRuleCheckExists.exists("grafana_rule.test_rule", &alertRule),
					resource.TestCheckResourceAttr("grafana_rule.test_rule", "name", name),
					resource.TestCheckResourceAttr("grafana_rule.test_rule", "rule_group", "My Rule Group"),
					resource.TestCheckResourceAttr("grafana_rule.test_rule", "for", "2m0s"),
					resource.TestCheckResourceAttr("grafana_rule.test_rule", "condition", "B"),
					resource.TestCheckResourceAttr("grafana_rule.test_rule", "no_data_state", "NoData"),
					resource.TestCheckResourceAttr("grafana_rule.test_rule", "exec_err_state", "Alerting"),
					resource.TestCheckResourceAttr("grafana_rule.test_rule", "is_paused", "false"),
					resource.TestCheckResourceAttr("grafana_rule.test_rule", "data.#", "2"),
					resource.TestCheckResourceAttr("grafana_rule.test_rule", "data.0.ref_id", "A"),
					resource.TestCheckResourceAttr("grafana_rule.test_rule", "data.0.datasource_uid", "PD8C576611E62080A"),
					resource.TestCheckResourceAttr("grafana_rule.test_rule", "data.1.ref_id", "B"),
					resource.TestCheckResourceAttr("grafana_rule.test_rule", "data.1.datasource_uid", "-100"),
					testutils.CheckLister("grafana_rule.test_rule"),
				),
			},
			// Test import.
			{
				ResourceName:      "grafana_rule.test_rule",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Test import without org ID.
			{
				ResourceName:      "grafana_rule.test_rule",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs := s.RootModule().Resources["grafana_rule.test_rule"]
					if rs == nil {
						return "", fmt.Errorf("resource not found")
					}
					return rs.Primary.Attributes["uid"], nil
				},
			},
			// Test update content.
			{
				Config: testAccAlertRuleBasic(name+"-updated", folderName),
				Check: resource.ComposeTestCheckFunc(
					alertingRuleCheckExists.exists("grafana_rule.test_rule", &alertRule),
					resource.TestCheckResourceAttr("grafana_rule.test_rule", "name", name+"-updated"),
					resource.TestCheckResourceAttr("grafana_rule.test_rule", "rule_group", "My Rule Group"),
					resource.TestCheckResourceAttr("grafana_rule.test_rule", "for", "2m0s"),
					resource.TestCheckResourceAttr("grafana_rule.test_rule", "condition", "B"),
					resource.TestCheckResourceAttr("grafana_rule.test_rule", "no_data_state", "NoData"),
					resource.TestCheckResourceAttr("grafana_rule.test_rule", "exec_err_state", "Alerting"),
					resource.TestCheckResourceAttr("grafana_rule.test_rule", "is_paused", "false"),
					resource.TestCheckResourceAttr("grafana_rule.test_rule", "data.#", "2"),
				),
			},
		},
	})
}

func TestAccAlertRule_organizationLifecycle(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t, ">=9.1.0")

	var alertRule models.ProvisionedAlertRule
	var org models.OrgDetailsDTO
	name := acctest.RandString(10)
	folderName := acctest.RandString(10)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             orgCheckExists.destroyed(&org, nil),
		Steps: []resource.TestStep{
			// Test creation.
			{
				Config: testAccAlertRuleInOrg(name, folderName),
				Check: resource.ComposeTestCheckFunc(
					alertingRuleCheckExists.exists("grafana_rule.test_rule", &alertRule),
					orgCheckExists.exists("grafana_organization.test", &org),
					checkResourceIsInOrg("grafana_rule.test_rule", "grafana_organization.test"),
					resource.TestCheckResourceAttr("grafana_rule.test_rule", "name", name),
					resource.TestCheckResourceAttr("grafana_rule.test_rule", "rule_group", "My Rule Group"),
					resource.TestCheckResourceAttr("grafana_rule.test_rule", "for", "2m0s"),
					resource.TestCheckResourceAttr("grafana_rule.test_rule", "condition", "B"),
					resource.TestCheckResourceAttr("grafana_rule.test_rule", "no_data_state", "NoData"),
					resource.TestCheckResourceAttr("grafana_rule.test_rule", "exec_err_state", "Alerting"),
					resource.TestCheckResourceAttr("grafana_rule.test_rule", "is_paused", "false"),
					resource.TestCheckResourceAttr("grafana_rule.test_rule", "data.#", "1"),
				),
			},
			// Test import.
			{
				ResourceName:      "grafana_rule.test_rule",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Test delete resource, but not org.
			{
				Config: testutils.WithoutResource(t, testAccAlertRuleInOrg(name, folderName), "grafana_rule.test_rule"),
				Check: resource.ComposeTestCheckFunc(
					orgCheckExists.exists("grafana_organization.test", &org),
					alertingRuleCheckExists.destroyed(&alertRule, &org),
				),
			},
		},
	})
}

func TestAccAlertRule_notificationSettings(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t, ">=10.4.0")

	var alertRule models.ProvisionedAlertRule
	name := acctest.RandString(10)
	folderName := acctest.RandString(10)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             alertingRuleCheckExists.destroyed(&alertRule, nil),
		Steps: []resource.TestStep{
			{
				Config: testAccAlertRuleWithNotificationSettings(name, folderName),
				Check: resource.ComposeTestCheckFunc(
					alertingRuleCheckExists.exists("grafana_rule.test_rule", &alertRule),
					resource.TestCheckResourceAttr("grafana_rule.test_rule", "name", name),
					resource.TestCheckResourceAttr("grafana_rule.test_rule", "notification_settings.#", "1"),
					resource.TestCheckResourceAttr("grafana_rule.test_rule", "notification_settings.0.contact_point", "test-contact-point"),
					resource.TestCheckResourceAttr("grafana_rule.test_rule", "notification_settings.0.group_wait", "45s"),
					resource.TestCheckResourceAttr("grafana_rule.test_rule", "notification_settings.0.group_interval", "6m"),
					resource.TestCheckResourceAttr("grafana_rule.test_rule", "notification_settings.0.repeat_interval", "3h"),
					resource.TestCheckResourceAttr("grafana_rule.test_rule", "notification_settings.0.mute_timings.#", "1"),
					resource.TestCheckResourceAttr("grafana_rule.test_rule", "notification_settings.0.mute_timings.0", "test-mute-timing"),
					resource.TestCheckResourceAttr("grafana_rule.test_rule", "notification_settings.0.group_by.#", "2"),
					resource.TestCheckResourceAttr("grafana_rule.test_rule", "notification_settings.0.group_by.0", "alertname"),
					resource.TestCheckResourceAttr("grafana_rule.test_rule", "notification_settings.0.group_by.1", "grafana_folder"),
				),
			},
		},
	})
}

// TODO: Add test for recording rule
// func TestAccAlertRule_recordingRule(t *testing.T) {
// 	testutils.CheckOSSTestsEnabled(t, ">=11.4.0")

// 	var alertRule models.ProvisionedAlertRule
// 	name := acctest.RandString(10)
// 	metric := "valid_metric"
// 	folderName := acctest.RandString(10)

// 	resource.ParallelTest(t, resource.TestCase{
// 		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
// 		CheckDestroy:             alertingRuleCheckExists.destroyed(&alertRule, nil),
// 		Steps: []resource.TestStep{
// 			{
// 				Config: testAccAlertRuleRecordingRule(name, metric, folderName),
// 				Check: resource.ComposeTestCheckFunc(
// 					alertingRuleCheckExists.exists("grafana_rule.test_rule", &alertRule),
// 					resource.TestCheckResourceAttr("grafana_rule.test_rule", "name", name),
// 					resource.TestCheckResourceAttr("grafana_rule.test_rule", "record.#", "1"),
// 					resource.TestCheckResourceAttr("grafana_rule.test_rule", "record.0.metric", metric),
// 					resource.TestCheckResourceAttr("grafana_rule.test_rule", "record.0.from", "A"),
// 					// ensure fields are empty as expected
// 					resource.TestCheckResourceAttr("grafana_rule.test_rule", "for", "0s"),
// 					resource.TestCheckResourceAttr("grafana_rule.test_rule", "condition", ""),
// 					resource.TestCheckResourceAttr("grafana_rule.test_rule", "no_data_state", ""),
// 					resource.TestCheckResourceAttr("grafana_rule.test_rule", "exec_err_state", ""),
// 				),
// 			},
// 		},
// 	})
// }

func TestAccAlertRule_moveToDifferentFolderAndGroup(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t, ">=9.1.0")

	var alertRule models.ProvisionedAlertRule
	name := acctest.RandString(10)
	folderUID := acctest.RandString(10)
	folderName := acctest.RandString(10)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             alertingRuleCheckExists.destroyed(&alertRule, nil),
		Steps: []resource.TestStep{
			// Test creation in initial folder and group
			{
				Config: testAccAlertRuleBasic(name, folderName),
				Check: resource.ComposeTestCheckFunc(
					alertingRuleCheckExists.exists("grafana_rule.test_rule", &alertRule),
					resource.TestCheckResourceAttr("grafana_rule.test_rule", "name", name),
					resource.TestCheckResourceAttr("grafana_rule.test_rule", "rule_group", "My Rule Group"),
				),
			},
			// Test moving to different folder and group
			{
				Config: testAccAlertRuleInDifferentFolderAndGroup(name, folderUID),
				Check: resource.ComposeTestCheckFunc(
					alertingRuleCheckExists.exists("grafana_rule.test_rule", &alertRule),
					resource.TestCheckResourceAttr("grafana_rule.test_rule", "name", name),
					resource.TestCheckResourceAttr("grafana_rule.test_rule", "rule_group", "New Rule Group"),
					resource.TestCheckResourceAttr("grafana_rule.test_rule", "folder_uid", folderUID),
				),
			},
		},
	})
}

func testAccAlertRuleBasic(name string, folderName string) string {
	return fmt.Sprintf(`
resource "grafana_folder" "rule_folder" {
  title = "%[2]s"
}

resource "grafana_rule" "test_rule" {
  name               = "%[1]s"
  folder_uid         = grafana_folder.rule_folder.uid
  rule_group         = "My Rule Group"
  for                = "2m"
  condition          = "B"
  no_data_state      = "NoData"
  exec_err_state     = "Alerting"
  annotations = {
    "a" = "b"
    "c" = "d"
  }
  labels = {
    "e" = "f"
    "g" = "h"
  }
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
  data {
    ref_id     = "B"
    query_type = ""
    relative_time_range {
      from = 0
      to   = 0
    }
    datasource_uid = "-100"
    model          = <<EOT
{
    "conditions": [
        {
        "evaluator": {
            "params": [
            3
            ],
            "type": "gt"
        },
        "operator": {
            "type": "and"
        },
        "query": {
            "params": [
            "A"
            ]
        },
        "reducer": {
            "params": [],
            "type": "last"
        },
        "type": "query"
        }
    ],
    "datasource": {
        "type": "__expr__",
        "uid": "-100"
    },
    "hide": false,
    "intervalMs": 1000,
    "maxDataPoints": 43200,
    "refId": "B",
    "type": "classic_conditions"
}
EOT
  }
}
`, name, folderName)
}

func testAccAlertRuleInOrg(name string, folderName string) string {
	return fmt.Sprintf(`
resource "grafana_organization" "test" {
  name = "%[1]s"
}

resource "grafana_folder" "rule_folder" {
  org_id = grafana_organization.test.id
  title  = "%[2]s"
}

resource "grafana_rule" "test_rule" {
  org_id             = grafana_organization.test.id
  name               = "%[1]s"
  folder_uid         = grafana_folder.rule_folder.uid
  rule_group         = "My Rule Group"
  for                = "2m"
  condition          = "B"
  no_data_state      = "NoData"
  exec_err_state     = "Alerting"
  is_paused          = false
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
`, name, folderName)
}

func testAccAlertRuleWithNotificationSettings(name string, folderName string) string {
	return fmt.Sprintf(`
resource "grafana_folder" "rule_folder" {
  title = "%[2]s"
}

resource "grafana_mute_timing" "test_mute_timing" {
  name = "test-mute-timing"
  intervals {}
}

resource "grafana_contact_point" "test_contact_point" {
  name = "test-contact-point"
  email {
    addresses = ["test@example.com"]
  }
}

resource "grafana_rule" "test_rule" {
  name               = "%[1]s"
  folder_uid         = grafana_folder.rule_folder.uid
  rule_group         = "My Rule Group"
  for                = "2m"
  condition          = "B"
  no_data_state      = "NoData"
  exec_err_state     = "Alerting"
  is_paused          = false
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
  notification_settings {
    contact_point    = grafana_contact_point.test_contact_point.name
    group_wait       = "45s"
    group_interval   = "6m"
    repeat_interval  = "3h"
    mute_timings     = [grafana_mute_timing.test_mute_timing.name]
    group_by         = ["alertname", "grafana_folder"]
  }
}
`, name, folderName)
}

// TODO: Add test for recording rule
// func testAccAlertRuleRecordingRule(name string, metric string, folderName string) string {
// 	return fmt.Sprintf(`
// resource "grafana_folder" "rule_folder" {
//   title = "%[3]s"
// }

// resource "grafana_data_source" "testdata_datasource" {
//   name = "%[1]s"
//   type = "grafana-testdata-datasource"
//   url  = "http://localhost:3333"
// }

// resource "grafana_rule" "test_rule" {
//   name               = "%[1]s"
//   folder_uid         = grafana_folder.rule_folder.uid
//   rule_group         = "My Rule Group"
//   is_paused          = false
//   data {
//     ref_id     = "A"
//     query_type = ""
//     relative_time_range {
//       from = 600
//       to   = 0
//     }
//     datasource_uid = grafana_data_source.testdata_datasource.uid
//     model = jsonencode({
//       intervalMs    = 1000
//       maxDataPoints = 43200
//       refId         = "A"
//     })
//   }
//   record {
//     metric = "%[2]s"
//     from   = "A"
//   }
// }
// `, name, metric, folderName)
// }

func testAccAlertRuleInDifferentFolderAndGroup(name string, folderUID string) string {
	return fmt.Sprintf(`
resource "grafana_folder" "new_folder" {
	title = "New Alert Rule Folder"
	uid   = "%[2]s"
}

resource "grafana_rule" "test_rule" {
	name               = "%[1]s"
	folder_uid         = grafana_folder.new_folder.uid
	rule_group         = "New Rule Group"
	for                = "2m"
	condition          = "B"
	no_data_state      = "NoData"
	exec_err_state     = "Alerting"
	is_paused          = false
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
`, name, folderUID)
}
