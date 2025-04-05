package grafana

import (
	"context"
	"strconv"
	"strings"

	"github.com/grafana/grafana-openapi-client-go/client/provisioning"
	"github.com/grafana/grafana-openapi-client-go/models"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/grafana/terraform-provider-grafana/v3/internal/common"
)

var resourceRuleGroupConfigID = common.NewResourceID(
	common.OptionalIntIDField("orgID"),
	common.StringIDField("folderUID"),
	common.StringIDField("ruleGroupName"),
)

func resourceRuleGroupConfig() *common.Resource {
	schema := &schema.Resource{
		Description: `
Manages Grafana Alerting rule group configuration.

* [Official documentation](https://grafana.com/docs/grafana/latest/alerting/set-up/provision-alerting-resources/terraform-provisioning/)
* [HTTP API](https://grafana.com/docs/grafana/latest/developers/http_api/alerting_provisioning/#alert-rules)

This resource requires Grafana 9.1.0 or later.
`,

		CreateContext: createRuleGroupConfig,
		ReadContext:   readRuleGroupConfig,
		UpdateContext: putRuleGroupConfig,
		DeleteContext: deleteRuleGroupConfig,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"org_id": orgIDAttribute(),
			"folder_uid": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				Description:  "The UID of the folder that the group belongs to.",
				ValidateFunc: folderUIDValidation,
			},
			"rule_group_name": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The name of the rule group.",
			},
			"interval_seconds": {
				Type:        schema.TypeInt,
				Required:    true,
				Description: "The interval, in seconds, at which all rules in the group are evaluated. If a group contains many rules, the rules are evaluated sequentially.",
			},
		},
	}

	return common.NewLegacySDKResource(
		common.CategoryAlerting,
		"grafana_rule_group_config",
		resourceRuleGroupConfigID,
		schema,
	)
}

func createRuleGroupConfig(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, orgID := OAPIClientFromNewOrgResource(meta, data)

	folderUID := data.Get("folder_uid").(string)
	ruleGroupName := data.Get("rule_group_name").(string)
	intervalSeconds := data.Get("interval_seconds").(int)

	// Create a new rule group with the specified configuration
	putParams := provisioning.NewPutAlertRuleGroupParams().
		WithFolderUID(folderUID).
		WithGroup(ruleGroupName).
		WithBody(&models.AlertRuleGroup{
			Title:     ruleGroupName,
			FolderUID: folderUID,
			Interval:  int64(intervalSeconds),
		})

	_, err := client.Provisioning.PutAlertRuleGroup(putParams)
	if err != nil {
		return diag.Errorf("failed to create alert rule group config: %v", err)
	}

	// Set the ID for the resource
	data.SetId(resourceRuleGroupConfigID.Make(orgID, folderUID, ruleGroupName))

	return readRuleGroupConfig(ctx, data, meta)
}

func readRuleGroupConfig(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, orgID, idWithoutOrg := OAPIClientFromExistingOrgResource(meta, data.Id())

	folderUID, name, found := strings.Cut(idWithoutOrg, common.ResourceIDSeparator)
	if !found {
		return diag.Errorf("invalid ID %q", idWithoutOrg)
	}

	resp, err := client.Provisioning.GetAlertRuleGroup(name, folderUID)
	if err, shouldReturn := common.CheckReadError("rule group", data, err); shouldReturn {
		return err
	}

	g := resp.Payload
	data.Set("rule_group_name", g.Title)
	data.Set("folder_uid", g.FolderUID)
	data.Set("interval_seconds", g.Interval)
	if orgIDRaw, ok := data.GetOk("org_id"); ok {
		data.Set("org_id", orgIDRaw)
	} else {
		data.Set("org_id", strconv.FormatInt(orgID, 10))
	}

	data.SetId(resourceRuleGroupID.Make(orgID, folderUID, name))

	return nil
}

func putRuleGroupConfig(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, _ := OAPIClientFromNewOrgResource(meta, data)

	folderUID := data.Get("folder_uid").(string)
	ruleGroupName := data.Get("rule_group_name").(string)
	intervalSeconds := data.Get("interval_seconds").(int)

	// Update the rule group configuration in Grafana
	putParams := provisioning.NewPutAlertRuleGroupParams().
		WithFolderUID(folderUID).
		WithGroup(ruleGroupName).
		WithBody(&models.AlertRuleGroup{
			Title:     ruleGroupName,
			FolderUID: folderUID,
			Interval:  int64(intervalSeconds),
		})

	_, err := client.Provisioning.PutAlertRuleGroup(putParams)
	if err != nil {
		return diag.Errorf("failed to update alert rule group config: %v", err)
	}

	return readRuleGroupConfig(ctx, data, meta)
}

func deleteRuleGroupConfig(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	// No need to delete anything as this resource doesn't represent an actual resource in Grafana.
	// It only configures properties of a rule group (like interval) which will be removed when the rule group itself is deleted.

	return nil
}
