package grafana

import (
	"context"
	"fmt"
	"strconv"

	gapi "github.com/grafana/grafana-api-golang-client"
	"github.com/grafana/terraform-provider-grafana/internal/common"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func ResourceTeamExternalGroup() *schema.Resource {
	return &schema.Resource{

		Description: "Use the `team_sync` attribute of the `grafana_team` resource instead.",

		CreateContext: CreateTeamExternalGroup,
		UpdateContext: UpdateTeamExternalGroup,
		DeleteContext: UpdateTeamExternalGroup,
		ReadContext:   ReadTeamExternalGroup,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		DeprecationMessage: "Use the `team_sync` attribute of the `grafana_team` resource instead.",

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
}

func CreateTeamExternalGroup(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	orgID, teamIDStr := SplitOrgResourceID(d.Get("team_id").(string))
	teamID, _ := strconv.ParseInt(teamIDStr, 10, 64)
	d.SetId(MakeOrgResourceID(orgID, teamID))
	client, _, _ := ClientFromExistingOrgResource(meta, d.Id())

	if err := manageTeamExternalGroup(client, teamID, d, "groups"); err != nil {
		return diag.FromErr(err)
	}

	return ReadTeamExternalGroup(ctx, d, meta)
}

func ReadTeamExternalGroup(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, orgID, idStr := ClientFromExistingOrgResource(meta, d.Id())
	teamID, _ := strconv.ParseInt(idStr, 10, 64)

	teamGroups, err := client.TeamGroups(teamID)
	if err, shouldReturn := common.CheckReadError("team groups", d, err); shouldReturn {
		return err
	}

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
	client, _, idStr := ClientFromExistingOrgResource(meta, d.Id())
	teamID, _ := strconv.ParseInt(idStr, 10, 64)

	if err := manageTeamExternalGroup(client, teamID, d, "groups"); err != nil {
		return diag.FromErr(err)
	}

	return ReadTeamExternalGroup(ctx, d, meta)
}

func manageTeamExternalGroup(client *gapi.Client, teamID int64, d *schema.ResourceData, groupsAttr string) error {
	addGroups, removeGroups := groupChangesTeamExternalGroup(d, groupsAttr)

	for _, group := range addGroups {
		err := client.NewTeamGroup(teamID, group)
		if err != nil {
			return fmt.Errorf("error adding group %s to team %d: %w", group, teamID, err)
		}
	}

	for _, group := range removeGroups {
		err := client.DeleteTeamGroup(teamID, group)
		if err != nil {
			return fmt.Errorf("error removing group %s from team %d: %w", group, teamID, err)
		}
	}

	return nil
}

func groupChangesTeamExternalGroup(d *schema.ResourceData, attr string) ([]string, []string) {
	// Get the lists of team groups read in from Grafana state (old) and configured (new)
	state, config := d.GetChange(attr)

	currentGroups := make([]string, state.(*schema.Set).Len())
	for i, v := range state.(*schema.Set).List() {
		currentGroups[i] = v.(string)
	}

	desiredGroups := make([]string, config.(*schema.Set).Len())
	for i, v := range config.(*schema.Set).List() {
		desiredGroups[i] = v.(string)
	}

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
