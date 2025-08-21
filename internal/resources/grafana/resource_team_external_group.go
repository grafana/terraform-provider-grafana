package grafana

import (
	"context"
	"fmt"
	"strconv"

	goapi "github.com/grafana/grafana-openapi-client-go/client"
	teamsSync "github.com/grafana/grafana-openapi-client-go/client/sync_team_groups"
	"github.com/grafana/grafana-openapi-client-go/models"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceTeamExternalGroup() *common.Resource {
	schema := &schema.Resource{
		Description: "Equivalent to the the `team_sync` attribute of the `grafana_team` resource. Use one or the other to configure a team's external groups syncing config.",

		CreateContext: CreateTeamExternalGroup,
		UpdateContext: UpdateTeamExternalGroup,
		DeleteContext: DeleteTeamExternalGroup,
		ReadContext:   ReadTeamExternalGroup,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"team_id": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The Team ID",
				DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
					_, old = SplitOrgResourceID(old)
					_, new = SplitOrgResourceID(new)
					return old == "0" && new == "" || old == "" && new == "0" || old == new
				},
			},

			"groups": {
				Type:     schema.TypeSet,
				Required: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Description: "The team external groups list",
			},
		},
	}

	return common.NewLegacySDKResource(
		common.CategoryGrafanaEnterprise,
		"grafana_team_external_group",
		orgResourceIDInt("teamID"),
		schema,
	)
}

func CreateTeamExternalGroup(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	orgID, teamIDStr := SplitOrgResourceID(d.Get("team_id").(string))
	teamID, _ := strconv.ParseInt(teamIDStr, 10, 64)
	d.SetId(MakeOrgResourceID(orgID, teamID))
	client, _, _ := OAPIClientFromExistingOrgResource(meta, d.Id())

	if err := manageTeamExternalGroup(client, teamID, d, "groups"); err != nil {
		return diag.FromErr(err)
	}

	return ReadTeamExternalGroup(ctx, d, meta)
}

func ReadTeamExternalGroup(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, orgID, idStr := OAPIClientFromExistingOrgResource(meta, d.Id())
	teamID, _ := strconv.ParseInt(idStr, 10, 64)

	resp, err := client.SyncTeamGroups.GetTeamGroupsAPI(teamID)
	if err, shouldReturn := common.CheckReadError("team groups", d, err); shouldReturn {
		return err
	}
	teamGroups := resp.GetPayload()

	groupIDs := make([]string, 0, len(teamGroups))
	for _, teamGroup := range teamGroups {
		groupIDs = append(groupIDs, teamGroup.GroupID)
	}
	d.SetId(MakeOrgResourceID(orgID, teamID))
	d.Set("team_id", d.Id())
	d.Set("groups", groupIDs)

	return diag.Diagnostics{}
}

func UpdateTeamExternalGroup(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, _, idStr := OAPIClientFromExistingOrgResource(meta, d.Id())
	teamID, _ := strconv.ParseInt(idStr, 10, 64)

	if err := manageTeamExternalGroup(client, teamID, d, "groups"); err != nil {
		return diag.FromErr(err)
	}

	return ReadTeamExternalGroup(ctx, d, meta)
}

func DeleteTeamExternalGroup(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, _, idStr := OAPIClientFromExistingOrgResource(meta, d.Id())
	teamID, _ := strconv.ParseInt(idStr, 10, 64)
	if err := applyTeamExternalGroup(client, teamID, nil, common.SetToStringSlice(d.Get("groups").(*schema.Set))); err != nil {
		return diag.FromErr(err)
	}
	return nil
}

func manageTeamExternalGroup(client *goapi.GrafanaHTTPAPI, teamID int64, d *schema.ResourceData, groupsAttr string) error {
	addGroups, removeGroups := groupChangesTeamExternalGroup(d, groupsAttr)
	return applyTeamExternalGroup(client, teamID, addGroups, removeGroups)
}

func applyTeamExternalGroup(client *goapi.GrafanaHTTPAPI, teamID int64, addGroups, removeGroups []string) error {
	for _, group := range addGroups {
		body := models.TeamGroupMapping{
			GroupID: group,
		}
		if _, err := client.SyncTeamGroups.AddTeamGroupAPI(teamID, &body); err != nil {
			return fmt.Errorf("error adding group %s to team %d: %w", group, teamID, err)
		}
	}

	for _, group := range removeGroups {
		params := teamsSync.NewRemoveTeamGroupAPIQueryParams().WithTeamID(teamID).WithGroupID(&group)
		if _, err := client.SyncTeamGroups.RemoveTeamGroupAPIQuery(params); err != nil {
			return fmt.Errorf("error removing group %s from team %d: %w", group, teamID, err)
		}
	}

	return nil
}

func groupChangesTeamExternalGroup(d *schema.ResourceData, attr string) ([]string, []string) {
	// Get the lists of team groups read in from Grafana state (old) and configured (new)
	state, config := d.GetChange(attr)

	currentGroups := common.SetToStringSlice(state.(*schema.Set))
	desiredGroups := common.SetToStringSlice(config.(*schema.Set))

	contains := func(slice []string, val string) bool {
		for _, item := range slice {
			if item == val {
				return true
			}
		}
		return false
	}

	addGroups := []string{}
	for _, group := range desiredGroups {
		if !contains(currentGroups, group) {
			addGroups = append(addGroups, group)
		}
	}
	removeGroups := []string{}
	for _, group := range currentGroups {
		if !contains(desiredGroups, group) {
			removeGroups = append(removeGroups, group)
		}
	}
	return addGroups, removeGroups
}
