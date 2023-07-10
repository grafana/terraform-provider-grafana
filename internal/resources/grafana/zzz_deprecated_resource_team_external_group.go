package grafana

import (
	"context"
	"fmt"
	"strconv"

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
				Type:        schema.TypeInt,
				Required:    true,
				ForceNew:    true,
				Description: "The Team ID",
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
	teamID := d.Get("team_id").(int)
	d.SetId(strconv.FormatInt(int64(teamID), 10))
	if err := manageTeamExternalGroup(d, meta, "groups"); err != nil {
		return diag.FromErr(err)
	}

	return diag.Diagnostics{}
}

func ReadTeamExternalGroup(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*common.Client).GrafanaAPI
	teamID, _ := strconv.ParseInt(d.Id(), 10, 64)
	teamGroups, err := client.TeamGroups(teamID)
	if err, shouldReturn := common.CheckReadError("team groups", d, err); shouldReturn {
		return err
	}

	groupIDs := make([]string, 0, len(teamGroups))
	for _, teamGroup := range teamGroups {
		groupIDs = append(groupIDs, teamGroup.GroupID)
	}
	d.Set("team_id", teamID)
	d.Set("groups", groupIDs)

	return diag.Diagnostics{}
}

func UpdateTeamExternalGroup(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	if err := manageTeamExternalGroup(d, meta, "groups"); err != nil {
		return diag.FromErr(err)
	}

	return diag.Diagnostics{}
}

func manageTeamExternalGroup(d *schema.ResourceData, meta interface{}, groupsAttr string) error {
	client := meta.(*common.Client).GrafanaAPI

	addGroups, removeGroups := groupChangesTeamExternalGroup(d, groupsAttr)
	teamID, _ := strconv.ParseInt(d.Id(), 10, 64)

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
