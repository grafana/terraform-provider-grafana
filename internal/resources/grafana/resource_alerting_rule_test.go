package grafana_test

import (
	"fmt"
	"testing"

	"github.com/grafana/grafana-openapi-client-go/models"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/grafana/terraform-provider-grafana/v4/internal/resources/grafana"
	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccRule_basic(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t, ">=9.1.0")

	var rule models.ProvisionedAlertRule

	expectedInitialConfig := `{"condition":"B","data":[{"datasourceUid":"__expr__","model":{"conditions":[{"evaluator":{"params":[0],"type":"gt"},"operator":{"type":"and"},"query":{"params":["A"]},"type":"query"}],"datasource":{"type":"__expr__","uid":"__expr__"},"expression":"A","reducer":"last","refId":"B","type":"classic_conditions"},"refId":"B","relativeTimeRange":{"from":600}}],"execErrState":"Error","folderUID":"test-folder","for":"0s","noDataState":"NoData","ruleGroup":"Test Group","title":"Test Rule","uid":"test-rule"}`
	expectedUpdatedTitleConfig := `{"condition":"B","data":[{"datasourceUid":"__expr__","model":{"conditions":[{"evaluator":{"params":[0],"type":"gt"},"operator":{"type":"and"},"query":{"params":["A"]},"type":"query"}],"datasource":{"type":"__expr__","uid":"__expr__"},"expression":"A","reducer":"last","refId":"B","type":"classic_conditions"},"refId":"B","relativeTimeRange":{"from":600}}],"execErrState":"Error","folderUID":"test-folder","for":"0s","noDataState":"NoData","ruleGroup":"Test Group","title":"Updated Rule Title","uid":"test-rule"}`
	expectedUpdatedUIDConfig := `{"condition":"B","data":[{"datasourceUid":"__expr__","model":{"conditions":[{"evaluator":{"params":[0],"type":"gt"},"operator":{"type":"and"},"query":{"params":["A"]},"type":"query"}],"datasource":{"type":"__expr__","uid":"__expr__"},"expression":"A","reducer":"last","refId":"B","type":"classic_conditions"},"refId":"B","relativeTimeRange":{"from":600}}],"execErrState":"Error","folderUID":"test-folder","for":"0s","noDataState":"NoData","ruleGroup":"Test Group","title":"Updated Rule Title","uid":"test-rule-updated"}`

	resource.Test(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             ruleCheckExists.destroyed(&rule, nil),
		Steps: []resource.TestStep{
			{
				// Test resource creation.
				Config: testutils.TestAccExample(t, "resources/grafana_alerting_rule/_acc_basic.tf"),
				Check: resource.ComposeTestCheckFunc(
					ruleCheckExists.exists("grafana_alerting_rule.test", &rule),
					resource.TestCheckResourceAttr("grafana_alerting_rule.test", "id", "1:test-rule"), // <org id>:<uid>
					resource.TestCheckResourceAttr("grafana_alerting_rule.test", "org_id", "1"),
					resource.TestCheckResourceAttr("grafana_alerting_rule.test", "uid", "test-rule"),
					resource.TestCheckResourceAttr(
						"grafana_alerting_rule.test", "config_json", expectedInitialConfig,
					),
					testutils.CheckLister("grafana_alerting_rule.test"),
				),
			},
			{
				// Updates title.
				Config: testutils.TestAccExample(t, "resources/grafana_alerting_rule/_acc_basic_update.tf"),
				Check: resource.ComposeTestCheckFunc(
					ruleCheckExists.exists("grafana_alerting_rule.test", &rule),
					resource.TestCheckResourceAttr("grafana_alerting_rule.test", "id", "1:test-rule"), // <org id>:<uid>
					resource.TestCheckResourceAttr("grafana_alerting_rule.test", "uid", "test-rule"),
					resource.TestCheckResourceAttr(
						"grafana_alerting_rule.test", "config_json", expectedUpdatedTitleConfig,
					),
				),
			},
			{
				// Updates uid.
				Config: testutils.TestAccExample(t, "resources/grafana_alerting_rule/_acc_basic_update_uid.tf"),
				Check: resource.ComposeTestCheckFunc(
					ruleCheckExists.exists("grafana_alerting_rule.test", &rule),
					resource.TestCheckResourceAttr("grafana_alerting_rule.test", "id", "1:test-rule-updated"), // <org id>:<uid>
					resource.TestCheckResourceAttr("grafana_alerting_rule.test", "uid", "test-rule-updated"),
					resource.TestCheckResourceAttr(
						"grafana_alerting_rule.test", "config_json", expectedUpdatedUIDConfig,
					),
				),
			},
			{
				// Importing matches the state of the previous step.
				ResourceName:            "grafana_alerting_rule.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{},
			},
		},
	})
}

func TestAccRule_uid_unset(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t, ">=9.1.0")

	var rule models.ProvisionedAlertRule

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             ruleCheckExists.destroyed(&rule, nil),
		Steps: []resource.TestStep{
			{
				// Create rule with no uid set.
				Config: testutils.TestAccExample(t, "resources/grafana_alerting_rule/_acc_uid_unset.tf"),
				Check: resource.ComposeTestCheckFunc(
					ruleCheckExists.exists("grafana_alerting_rule.test", &rule),
					resource.TestMatchResourceAttr("grafana_alerting_rule.test", "uid", common.UIDRegexp),
					// Config JSON should not contain uid
					resource.TestCheckResourceAttrWith("grafana_alerting_rule.test", "config_json", func(value string) error {
						ruleMap, err := grafana.UnmarshalConfigJSON(value)
						if err != nil {
							return err
						}
						if _, ok := ruleMap["uid"]; ok {
							return fmt.Errorf("uid should not be in config_json when not set")
						}
						return nil
					}),
				),
			},
			{
				// Update it to add a uid. We want to ensure that this causes a diff
				// and subsequent update.
				Config: testutils.TestAccExample(t, "resources/grafana_alerting_rule/_acc_uid_unset_set.tf"),
				Check: resource.ComposeTestCheckFunc(
					ruleCheckExists.exists("grafana_alerting_rule.test", &rule),
					resource.TestCheckResourceAttr("grafana_alerting_rule.test", "uid", "uid-previously-unset"),
					resource.TestCheckResourceAttrWith("grafana_alerting_rule.test", "config_json", func(value string) error {
						ruleMap, err := grafana.UnmarshalConfigJSON(value)
						if err != nil {
							return err
						}
						if uid, ok := ruleMap["uid"].(string); !ok || uid != "uid-previously-unset" {
							return fmt.Errorf("uid should be 'uid-previously-unset' in config_json, got: %v", ruleMap["uid"])
						}
						return nil
					}),
				),
			},
			{
				// Remove the uid once again to ensure this is also supported.
				Config: testutils.TestAccExample(t, "resources/grafana_alerting_rule/_acc_uid_unset.tf"),
				Check: resource.ComposeTestCheckFunc(
					ruleCheckExists.exists("grafana_alerting_rule.test", &rule),
					resource.TestCheckResourceAttrWith("grafana_alerting_rule.test", "config_json", func(value string) error {
						ruleMap, err := grafana.UnmarshalConfigJSON(value)
						if err != nil {
							return err
						}
						if _, ok := ruleMap["uid"]; ok {
							return fmt.Errorf("uid should not be in config_json when not set")
						}
						return nil
					}),
				),
			},
		},
	})
}

func TestAccRule_inOrg(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t, ">=9.1.0")

	var rule models.ProvisionedAlertRule
	var folder models.Folder
	var org models.OrgDetailsDTO

	orgName := acctest.RandString(10)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy: resource.ComposeTestCheckFunc(
			ruleCheckExists.destroyed(&rule, &org),
			folderCheckExists.destroyed(&folder, &org),
		),
		Steps: []resource.TestStep{
			{
				Config: testAccRuleInOrganization(orgName),
				Check: resource.ComposeTestCheckFunc(
					orgCheckExists.exists("grafana_organization.test", &org),
					// Check that the folder is in the correct organization
					folderCheckExists.exists("grafana_folder.test", &folder),
					resource.TestCheckResourceAttr("grafana_folder.test", "uid", "folder-"+orgName),
					resource.TestMatchResourceAttr("grafana_folder.test", "id", nonDefaultOrgIDRegexp),
					checkResourceIsInOrg("grafana_folder.test", "grafana_organization.test"),

					// Check that the rule is in the correct organization
					ruleCheckExists.exists("grafana_alerting_rule.test", &rule),
					resource.TestCheckResourceAttr("grafana_alerting_rule.test", "uid", "rule-"+orgName),
					resource.TestMatchResourceAttr("grafana_alerting_rule.test", "id", nonDefaultOrgIDRegexp),
					checkResourceIsInOrg("grafana_alerting_rule.test", "grafana_organization.test"),

					testAccRuleCheckExistsInFolder(&rule, &folder),
					testutils.CheckLister("grafana_alerting_rule.test"),
				),
			},
		},
	})
}

func testAccRuleCheckExistsInFolder(rule *models.ProvisionedAlertRule, folder *models.Folder) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if *rule.FolderUID != folder.UID && folder.UID != "" {
			return fmt.Errorf("rule.FolderUID(%s) does not match folder.UID(%s)", *rule.FolderUID, folder.UID)
		}
		return nil
	}
}

func Test_NormalizeConfigJSON(t *testing.T) {
	testutils.IsUnitTest(t)

	type args struct {
		config interface{}
	}

	title := "Test Rule"
	expected := fmt.Sprintf("{\"condition\":\"B\",\"data\":[{\"datasourceUid\":\"__expr__\",\"model\":{\"refId\":\"B\"},\"refId\":\"B\",\"relativeTimeRange\":{\"from\":600}}],\"execErrState\":\"Error\",\"folderUID\":\"test\",\"for\":\"0s\",\"noDataState\":\"NoData\",\"ruleGroup\":\"Test\",\"title\":\"%s\"}", title)

	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "String rule is valid",
			args: args{config: fmt.Sprintf("{\"title\":\"%s\",\"condition\":\"B\",\"data\":[{\"refId\":\"B\",\"datasourceUid\":\"__expr__\",\"model\":{\"refId\":\"B\"},\"relativeTimeRange\":{\"from\":600,\"to\":0}}],\"execErrState\":\"Error\",\"noDataState\":\"NoData\",\"folderUID\":\"test\",\"ruleGroup\":\"Test\",\"for\":\"0s\"}", title)},
			want: expected,
		},
		{
			name: "Map rule is valid",
			args: args{config: map[string]interface{}{
				"title":        title,
				"condition":    "B",
				"data":         []interface{}{map[string]interface{}{"refId": "B", "datasourceUid": "__expr__", "model": map[string]interface{}{"refId": "B"}, "relativeTimeRange": map[string]interface{}{"from": float64(600), "to": float64(0)}}},
				"execErrState": "Error",
				"noDataState":  "NoData",
				"folderUID":    "test",
				"ruleGroup":    "Test",
				"for":          "0s",
			}},
			want: expected,
		},
		{
			name: "OrgID is removed",
			args: args{config: map[string]interface{}{
				"title":        title,
				"condition":    "B",
				"data":         []interface{}{map[string]interface{}{"refId": "B", "datasourceUid": "__expr__", "model": map[string]interface{}{"refId": "B"}, "relativeTimeRange": map[string]interface{}{"from": float64(600), "to": float64(0)}}},
				"execErrState": "Error",
				"noDataState":  "NoData",
				"folderUID":    "test",
				"ruleGroup":    "Test",
				"for":          "0s",
				"orgID":        float64(1),
			}},
			want: expected,
		},
		{
			name: "Provenance is removed",
			args: args{config: map[string]interface{}{
				"title":        title,
				"condition":    "B",
				"data":         []interface{}{map[string]interface{}{"refId": "B", "datasourceUid": "__expr__", "model": map[string]interface{}{"refId": "B"}, "relativeTimeRange": map[string]interface{}{"from": float64(600), "to": float64(0)}}},
				"execErrState": "Error",
				"noDataState":  "NoData",
				"folderUID":    "test",
				"ruleGroup":    "Test",
				"for":          "0s",
				"provenance":   "api",
			}},
			want: expected,
		},
		{
			name: "Default maxDataPoints is removed",
			args: args{config: map[string]interface{}{
				"title":        title,
				"condition":    "B",
				"data":         []interface{}{map[string]interface{}{"refId": "B", "datasourceUid": "__expr__", "model": map[string]interface{}{"refId": "B", "maxDataPoints": float64(43200)}, "relativeTimeRange": map[string]interface{}{"from": float64(600), "to": float64(0)}}},
				"execErrState": "Error",
				"noDataState":  "NoData",
				"folderUID":    "test",
				"ruleGroup":    "Test",
				"for":          "0s",
			}},
			want: expected,
		},
		{
			name: "Default intervalMs is removed",
			args: args{config: map[string]interface{}{
				"title":        title,
				"condition":    "B",
				"data":         []interface{}{map[string]interface{}{"refId": "B", "datasourceUid": "__expr__", "model": map[string]interface{}{"refId": "B", "intervalMs": float64(1000)}, "relativeTimeRange": map[string]interface{}{"from": float64(600), "to": float64(0)}}},
				"execErrState": "Error",
				"noDataState":  "NoData",
				"folderUID":    "test",
				"ruleGroup":    "Test",
				"for":          "0s",
			}},
			want: expected,
		},
		{
			name: "Duration 0s is normalized to 0s",
			args: args{config: map[string]interface{}{
				"title":        title,
				"condition":    "B",
				"data":         []interface{}{map[string]interface{}{"refId": "B", "datasourceUid": "__expr__", "model": map[string]interface{}{"refId": "B"}, "relativeTimeRange": map[string]interface{}{"from": float64(600), "to": float64(0)}}},
				"execErrState": "Error",
				"noDataState":  "NoData",
				"folderUID":    "test",
				"ruleGroup":    "Test",
				"for":          "0s",
			}},
			want: expected,
		},
		{
			name: "Duration 5m0s is normalized to 300s",
			args: args{config: map[string]interface{}{
				"title":        title,
				"condition":    "B",
				"data":         []interface{}{map[string]interface{}{"refId": "B", "datasourceUid": "__expr__", "model": map[string]interface{}{"refId": "B"}, "relativeTimeRange": map[string]interface{}{"from": float64(600), "to": float64(0)}}},
				"execErrState": "Error",
				"noDataState":  "NoData",
				"folderUID":    "test",
				"ruleGroup":    "Test",
				"for":          "5m0s",
			}},
			want: fmt.Sprintf("{\"condition\":\"B\",\"data\":[{\"datasourceUid\":\"__expr__\",\"model\":{\"refId\":\"B\"},\"refId\":\"B\",\"relativeTimeRange\":{\"from\":600}}],\"execErrState\":\"Error\",\"folderUID\":\"test\",\"for\":\"300s\",\"noDataState\":\"NoData\",\"ruleGroup\":\"Test\",\"title\":\"%s\"}", title),
		},
		{
			name: "RelativeTimeRange without 'to' field is preserved",
			args: args{config: map[string]interface{}{
				"title":        title,
				"condition":    "B",
				"data":         []interface{}{map[string]interface{}{"refId": "B", "datasourceUid": "__expr__", "model": map[string]interface{}{"refId": "B"}, "relativeTimeRange": map[string]interface{}{"from": float64(600)}}},
				"execErrState": "Error",
				"noDataState":  "NoData",
				"folderUID":    "test",
				"ruleGroup":    "Test",
				"for":          "0m",
			}},
			want: fmt.Sprintf("{\"condition\":\"B\",\"data\":[{\"datasourceUid\":\"__expr__\",\"model\":{\"refId\":\"B\"},\"refId\":\"B\",\"relativeTimeRange\":{\"from\":600}}],\"execErrState\":\"Error\",\"folderUID\":\"test\",\"for\":\"0s\",\"noDataState\":\"NoData\",\"ruleGroup\":\"Test\",\"title\":\"%s\"}", title),
		},
		{
			name: "Bad json is ignored",
			args: args{config: "74D93920-ED26–11E3-AC10–0800200C9A66"},
			want: "74D93920-ED26–11E3-AC10–0800200C9A66",
		},
		{
			name: "Notification settings null arrays are removed",
			args: args{config: map[string]interface{}{
				"title":        title,
				"condition":    "B",
				"data":         []interface{}{map[string]interface{}{"refId": "B", "datasourceUid": "__expr__", "model": map[string]interface{}{"refId": "B"}, "relativeTimeRange": map[string]interface{}{"from": float64(600), "to": float64(0)}}},
				"execErrState": "Error",
				"noDataState":  "NoData",
				"folderUID":    "test",
				"ruleGroup":    "Test",
				"for":          "0s",
				"notification_settings": map[string]interface{}{
					"receiver":              "test-receiver",
					"mute_time_intervals":   nil,
					"active_time_intervals": nil,
					"group_by":              nil,
				},
			}},
			want: fmt.Sprintf("{\"condition\":\"B\",\"data\":[{\"datasourceUid\":\"__expr__\",\"model\":{\"refId\":\"B\"},\"refId\":\"B\",\"relativeTimeRange\":{\"from\":600}}],\"execErrState\":\"Error\",\"folderUID\":\"test\",\"for\":\"0s\",\"noDataState\":\"NoData\",\"notification_settings\":{\"receiver\":\"test-receiver\"},\"ruleGroup\":\"Test\",\"title\":\"%s\"}", title),
		},
		{
			name: "Notification settings empty arrays are removed",
			args: args{config: map[string]interface{}{
				"title":        title,
				"condition":    "B",
				"data":         []interface{}{map[string]interface{}{"refId": "B", "datasourceUid": "__expr__", "model": map[string]interface{}{"refId": "B"}, "relativeTimeRange": map[string]interface{}{"from": float64(600), "to": float64(0)}}},
				"execErrState": "Error",
				"noDataState":  "NoData",
				"folderUID":    "test",
				"ruleGroup":    "Test",
				"for":          "0s",
				"notification_settings": map[string]interface{}{
					"receiver":              "test-receiver",
					"mute_time_intervals":   []interface{}{},
					"active_time_intervals": []interface{}{},
					"group_by":              []interface{}{},
				},
			}},
			want: fmt.Sprintf("{\"condition\":\"B\",\"data\":[{\"datasourceUid\":\"__expr__\",\"model\":{\"refId\":\"B\"},\"refId\":\"B\",\"relativeTimeRange\":{\"from\":600}}],\"execErrState\":\"Error\",\"folderUID\":\"test\",\"for\":\"0s\",\"noDataState\":\"NoData\",\"notification_settings\":{\"receiver\":\"test-receiver\"},\"ruleGroup\":\"Test\",\"title\":\"%s\"}", title),
		},
		{
			name: "Notification settings non-empty arrays are preserved",
			args: args{config: map[string]interface{}{
				"title":        title,
				"condition":    "B",
				"data":         []interface{}{map[string]interface{}{"refId": "B", "datasourceUid": "__expr__", "model": map[string]interface{}{"refId": "B"}, "relativeTimeRange": map[string]interface{}{"from": float64(600), "to": float64(0)}}},
				"execErrState": "Error",
				"noDataState":  "NoData",
				"folderUID":    "test",
				"ruleGroup":    "Test",
				"for":          "0s",
				"notification_settings": map[string]interface{}{
					"receiver":            "test-receiver",
					"mute_time_intervals": []interface{}{"maintenance-window"},
					"group_by":            []interface{}{"alertname", "grafana_folder"},
				},
			}},
			want: fmt.Sprintf("{\"condition\":\"B\",\"data\":[{\"datasourceUid\":\"__expr__\",\"model\":{\"refId\":\"B\"},\"refId\":\"B\",\"relativeTimeRange\":{\"from\":600}}],\"execErrState\":\"Error\",\"folderUID\":\"test\",\"for\":\"0s\",\"noDataState\":\"NoData\",\"notification_settings\":{\"group_by\":[\"alertname\",\"grafana_folder\"],\"mute_time_intervals\":[\"maintenance-window\"],\"receiver\":\"test-receiver\"},\"ruleGroup\":\"Test\",\"title\":\"%s\"}", title),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := grafana.NormalizeConfigJSON(tt.args.config); got != tt.want {
				t.Errorf("NormalizeConfigJSON() = %v, want %v", got, tt.want)
			}
		})
	}
}

func testAccRuleInOrganization(orgName string) string {
	return fmt.Sprintf(`
resource "grafana_organization" "test" {
	name = "%[1]s"
}

resource "grafana_folder" "test" {
	org_id  = grafana_organization.test.id
	title   = "folder-%[1]s"
	uid     = "folder-%[1]s"
}

resource "grafana_alerting_rule" "test" {
	org_id      = grafana_organization.test.id
	config_json = jsonencode({
		title        = "rule-%[1]s"
		uid          = "rule-%[1]s"
		condition    = "B"
		folderUID    = grafana_folder.test.uid
		ruleGroup    = "Test Group"
		noDataState  = "NoData"
		execErrState = "Error"
		for          = "0m"
		data = [
			{
				refId         = "B"
				datasourceUid = "__expr__"
				model = {
					refId = "B"
					type  = "classic_conditions"
					conditions = [
						{
							type     = "query"
							operator = { type = "and" }
							query    = { params = ["A"] }
							evaluator = {
								type   = "gt"
								params = [0]
							}
						}
					]
					datasource = {
						type = "__expr__"
						uid  = "__expr__"
					}
					expression = "A"
					reducer    = "last"
				}
				relativeTimeRange = {
					from = 600
					to   = 0
				}
			}
		]
	})
}`, orgName)
}
