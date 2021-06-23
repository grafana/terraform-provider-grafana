package grafana

import (
	"context"
	"strconv"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func ResourceTeamExternalGroup() *schema.Resource {
	return &schema.Resource{

		Description: `
* [Official documentation](https://grafana.com/docs/grafana/latest/enterprise/team-sync/)
* [HTTP API](https://grafana.com/docs/grafana/latest/http_api/external_group_sync/)
`,

		CreateContext: CreateTeamExternalGroup,
		UpdateContext: UpdateTeamExternalGroup,
		DeleteContext: DeleteTeamExternalGroup,
		ReadContext:   ReadTeamExternalGroup,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

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
	if err := UpdateGroups(d, meta); err != nil {
		return diag.FromErr(err)
	}

	return diag.Diagnostics{}
}

func ReadTeamExternalGroup(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	if err := ReadGroups(d, meta); err != nil {
		return diag.FromErr(err)
	}
	return nil
}

func UpdateTeamExternalGroup(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	if err := UpdateGroups(d, meta); err != nil {
		return diag.FromErr(err)
	}

	return diag.Diagnostics{}
}

func DeleteTeamExternalGroup(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	if err := UpdateGroups(d, meta); err != nil {
		return diag.FromErr(err)
	}
	d.SetId("")

	return diag.Diagnostics{}
}

func ReadGroups(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*client).gapi
	teamID, _ := strconv.ParseInt(d.Id(), 10, 64)
	teamGroups, err := client.TeamGroups(teamID)
	if err != nil {
		return err
	}
	
	groupIDs := make([]string, 0, len(teamGroups))
	for _, teamGroup := range teamGroups {
		groupIDs = append(groupIDs, teamGroup.GroupID)
	}
	d.Set("groups", groupIDs)

	return nil
}

func UpdateGroups(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*client).gapi

	addGroups, removeGroups := groupChanges(d)
	teamID, _ := strconv.ParseInt(d.Id(), 10, 64)

	for _, group := range addGroups {
		err := client.NewTeamGroup(teamID, group)
		if err != nil {
			return err
		}
	}

	for _, group := range removeGroups {
		err := client.DeleteTeamGroup(teamID, group)
		if err != nil {
			return err
		}
	}

	return nil
}

func groupChanges(d *schema.ResourceData) ([]string, []string) {
	// Get the lists of team groups read in from Grafana state (old) and configured (new)
	state, config := d.GetChange("groups")

	stateGroups := make([]string, state.(*schema.Set).Len())
	for i, v := range state.(*schema.Set).List() {
		stateGroups[i] = v.(string)
	}

	configGroups := make([]string, config.(*schema.Set).Len())
	for i, v := range config.(*schema.Set).List() {
		configGroups[i] = v.(string)
	}

	addGroups := []string{}
	for _, group := range configGroups {
		if !SliceFind(stateGroups, group) {
			addGroups = append(addGroups, group)
		}
	}

	removeGroups := []string{}
	for _, group := range stateGroups {
		if !SliceFind(configGroups, group) {
			removeGroups = append(removeGroups, group)
		}
	}

	return addGroups, removeGroups
}

// SliceFind takes a slice and looks for an element in it. If found it will
// return true, otherwise false.
func SliceFind(slice []string, val string) bool {
	for _, item := range slice {
		if item == val {
			return true
		}
	}
	return false
}

func RemoveIndex(s []string, index int) []string {
	return append(s[:index], s[index+1:]...)
}
