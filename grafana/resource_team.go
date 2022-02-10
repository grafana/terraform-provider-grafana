package grafana

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

type TeamMember struct {
	ID    int64
	Email string
}

type MemberChange struct {
	Type   ChangeMemberType
	Member TeamMember
}

type ChangeMemberType int8

const (
	AddMember ChangeMemberType = iota
	RemoveMember
)

func ResourceTeam() *schema.Resource {
	return &schema.Resource{

		Description: `
* [Official documentation](https://grafana.com/docs/grafana/latest/manage-users/manage-teams/)
* [HTTP API](https://grafana.com/docs/grafana/latest/http_api/team/)
`,

		CreateContext: CreateTeam,
		ReadContext:   ReadTeam,
		UpdateContext: UpdateTeam,
		DeleteContext: DeleteTeam,
		Exists:        ExistsTeam,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"team_id": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "The team id assigned to this team by Grafana.",
			},
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The display name for the Grafana team created.",
			},
			"email": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "An email address for the team.",
			},
			"members": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Description: `
A set of email addresses corresponding to users who should be given membership
to the team. Note: users specified here must already exist in Grafana.
`,
			},
		},
	}
}

func CreateTeam(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*client).gapi
	name := d.Get("name").(string)
	email := d.Get("email").(string)
	teamID, err := client.AddTeam(name, email)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(strconv.FormatInt(teamID, 10))
	d.Set("team_id", teamID)
	if err = UpdateMembers(d, meta); err != nil {
		return diag.FromErr(err)
	}

	return diag.Diagnostics{}
}

func ReadTeam(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*client).gapi
	teamID, _ := strconv.ParseInt(d.Id(), 10, 64)
	resp, err := client.Team(teamID)
	if err != nil && strings.HasPrefix(err.Error(), "status: 404") {
		log.Printf("[WARN] removing team %s from state because it no longer exists in grafana", d.Id())
		d.SetId("")
		return nil
	}
	if err != nil {
		return diag.FromErr(err)
	}
	d.Set("name", resp.Name)
	if resp.Email != "" {
		d.Set("email", resp.Email)
	}
	if err := ReadMembers(d, meta); err != nil {
		return diag.FromErr(err)
	}
	return nil
}

func UpdateTeam(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*client).gapi
	teamID, _ := strconv.ParseInt(d.Id(), 10, 64)
	if d.HasChange("name") || d.HasChange("email") {
		name := d.Get("name").(string)
		email := d.Get("email").(string)
		err := client.UpdateTeam(teamID, name, email)
		if err != nil {
			return diag.FromErr(err)
		}
	}
	if err := UpdateMembers(d, meta); err != nil {
		return diag.FromErr(err)
	}

	return diag.Diagnostics{}
}

func DeleteTeam(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*client).gapi
	teamID, _ := strconv.ParseInt(d.Id(), 10, 64)
	if err := client.DeleteTeam(teamID); err != nil {
		return diag.FromErr(err)
	}

	return diag.Diagnostics{}
}

func ExistsTeam(d *schema.ResourceData, meta interface{}) (bool, error) {
	client := meta.(*client).gapi
	teamID, _ := strconv.ParseInt(d.Id(), 10, 64)
	_, err := client.Team(teamID)
	if err != nil && strings.HasPrefix(err.Error(), "status: 404") {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, err
}

func ReadMembers(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*client).gapi
	teamID, _ := strconv.ParseInt(d.Id(), 10, 64)
	teamMembers, err := client.TeamMembers(teamID)
	if err != nil {
		return err
	}
	memberSlice := []string{}
	for _, teamMember := range teamMembers {
		memberSlice = append(memberSlice, teamMember.Email)
	}
	d.Set("members", memberSlice)

	return nil
}

func UpdateMembers(d *schema.ResourceData, meta interface{}) error {
	stateMembers, configMembers, err := collectMembers(d)
	if err != nil {
		return err
	}
	// compile the list of differences between current state and config
	changes := memberChanges(stateMembers, configMembers)
	// retrieves the corresponding user IDs based on the email provided
	changes, err = addMemberIdsToChanges(meta, changes)
	if err != nil {
		return err
	}
	teamID, _ := strconv.ParseInt(d.Id(), 10, 64)
	// now we can make the corresponding updates so current state matches config
	return applyMemberChanges(meta, teamID, changes)
}

func collectMembers(d *schema.ResourceData) (map[string]TeamMember, map[string]TeamMember, error) {
	stateMembers, configMembers := make(map[string]TeamMember), make(map[string]TeamMember)

	// Get the lists of team members read in from Grafana state (old) and configured (new)
	state, config := d.GetChange("members")
	for _, u := range state.(*schema.Set).List() {
		login := u.(string)
		// Sanity check that a member isn't specified twice within a team
		if _, ok := stateMembers[login]; ok {
			return nil, nil, fmt.Errorf("error: Team Member '%s' cannot be specified multiple times", login)
		}
		stateMembers[login] = TeamMember{0, login}
	}
	for _, u := range config.(*schema.Set).List() {
		login := u.(string)
		// Sanity check that a member isn't specified twice within a team
		if _, ok := configMembers[login]; ok {
			return nil, nil, fmt.Errorf("error: Team Member '%s' cannot be specified multiple times", login)
		}
		configMembers[login] = TeamMember{0, login}
	}

	return stateMembers, configMembers, nil
}

func memberChanges(stateMembers, configMembers map[string]TeamMember) []MemberChange {
	var changes []MemberChange
	for _, user := range configMembers {
		_, ok := stateMembers[user.Email]
		if !ok {
			// Member doesn't exist in Grafana's state for the team, should be added.
			changes = append(changes, MemberChange{AddMember, user})
			continue
		}
	}
	for _, user := range stateMembers {
		if _, ok := configMembers[user.Email]; !ok {
			// Member exists in Grafana's state for the team, but isn't
			// present in the team configuration, should be removed.
			changes = append(changes, MemberChange{RemoveMember, user})
		}
	}
	return changes
}

func addMemberIdsToChanges(meta interface{}, changes []MemberChange) ([]MemberChange, error) {
	client := meta.(*client).gapi
	gUserMap := make(map[string]int64)
	gUsers, err := client.OrgUsersCurrent()
	if err != nil {
		return nil, err
	}
	for _, u := range gUsers {
		gUserMap[u.Email] = u.UserID
	}
	var output []MemberChange

	for _, change := range changes {
		id, ok := gUserMap[change.Member.Email]
		if !ok {
			return nil, fmt.Errorf("error adding user %s. User does not exist in Grafana", change.Member.Email)
		}

		change.Member.ID = id
		output = append(output, change)
	}
	return output, nil
}

func applyMemberChanges(meta interface{}, teamID int64, changes []MemberChange) error {
	var err error
	client := meta.(*client).gapi
	for _, change := range changes {
		u := change.Member
		switch change.Type {
		case AddMember:
			err = client.AddTeamMember(teamID, u.ID)
		case RemoveMember:
			err = client.RemoveMemberFromTeam(teamID, u.ID)
		}
		if err != nil {
			return err
		}
	}
	return nil
}
