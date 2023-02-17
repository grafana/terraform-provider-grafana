package grafana

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"

	gapi "github.com/grafana/grafana-api-golang-client"
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
* [Official documentation](https://grafana.com/docs/grafana/latest/administration/team-management/)
* [HTTP API](https://grafana.com/docs/grafana/latest/developers/http_api/team/)
`,

		CreateContext: createTeam,
		ReadContext:   readTeam,
		UpdateContext: updateTeam,
		DeleteContext: deleteTeam,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"org_id": orgIDAttribute(),
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
				DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
					if (new == "[]" && old == "") || (new == "" && old == "[]") {
						return true
					}
					return false
				},
			},
			"ignore_externally_synced_members": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
				Description: `
Ignores team members that have been added to team by [Team Sync](https://grafana.com/docs/grafana/latest/setup-grafana/configure-security/configure-team-sync/).
Team Sync can be provisioned using [grafana_team_external_group resource](https://registry.terraform.io/providers/grafana/grafana/latest/docs/resources/team_external_group).
`,
			},
		},
	}
}

func createTeam(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, orgID := ClientFromNewOrgResource(meta, d)

	name := d.Get("name").(string)
	email := d.Get("email").(string)
	teamID, err := client.AddTeam(name, email)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(MakeOrgResourceID(orgID, strconv.FormatInt(teamID, 10)))

	if err := updateTeamMembers(d, meta); err != nil {
		return err
	}

	return readTeam(ctx, d, meta)
}

func readTeam(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, _, teamIDStr := ClientFromExistingOrgResource(meta, d.Id())
	teamID, _ := strconv.ParseInt(teamIDStr, 10, 64)
	resp, err := client.Team(teamID)
	if err != nil && strings.HasPrefix(err.Error(), "status: 404") {
		log.Printf("[WARN] removing team %s from state because it no longer exists in grafana", d.Id())
		d.SetId("")
		return nil
	}
	if err != nil {
		return diag.FromErr(err)
	}
	d.Set("team_id", teamID)
	d.Set("name", resp.Name)
	if resp.Email != "" {
		d.Set("email", resp.Email)
	}
	d.Set("org_id", strconv.FormatInt(resp.OrgID, 10))

	return readTeamMembers(d, meta)
}

func updateTeam(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, _, teamIDStr := ClientFromExistingOrgResource(meta, d.Id())
	teamID, _ := strconv.ParseInt(teamIDStr, 10, 64)
	if d.HasChange("name") || d.HasChange("email") {
		name := d.Get("name").(string)
		email := d.Get("email").(string)
		err := client.UpdateTeam(teamID, name, email)
		if err != nil {
			return diag.FromErr(err)
		}
	}

	if err := updateTeamMembers(d, meta); err != nil {
		return err
	}

	return readTeam(ctx, d, meta)
}

func deleteTeam(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, _, teamIDStr := ClientFromExistingOrgResource(meta, d.Id())
	teamID, _ := strconv.ParseInt(teamIDStr, 10, 64)
	return diag.FromErr(client.DeleteTeam(teamID))
}

func readTeamMembers(d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, _, teamIDStr := ClientFromExistingOrgResource(meta, d.Id())
	teamID, _ := strconv.ParseInt(teamIDStr, 10, 64)
	teamMembers, err := client.TeamMembers(teamID)
	if err != nil {
		return diag.FromErr(err)
	}
	memberSlice := []string{}
	for _, teamMember := range teamMembers {
		// Admin is added automatically to the team when the team is created.
		// We can't interact with it, so we skip it from Terraform management.
		if teamMember.Email == "admin@localhost" {
			continue
		}
		// Labels store information about auth provider used to sync the team member.
		// Team synced members should be managed through team_external_group resource and should be ignored here.
		ignoreExternallySynced, hasKey := d.GetOk("ignore_externally_synced_members")
		if (!hasKey || ignoreExternallySynced.(bool)) && len(teamMember.Labels) > 0 {
			continue
		}
		memberSlice = append(memberSlice, teamMember.Email)
	}
	d.Set("members", memberSlice)

	return nil
}

func updateTeamMembers(d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, _, teamIDStr := ClientFromExistingOrgResource(meta, d.Id())
	teamID, _ := strconv.ParseInt(teamIDStr, 10, 64)
	stateMembers, configMembers, err := collectMembers(d)
	if err != nil {
		return diag.FromErr(err)
	}
	// compile the list of differences between current state and config
	changes := memberChanges(stateMembers, configMembers)
	// retrieves the corresponding user IDs based on the email provided
	changes, err = addMemberIdsToChanges(client, changes)
	if err != nil {
		return diag.FromErr(err)
	}
	// now we can make the corresponding updates so current state matches config
	return applyMemberChanges(client, teamID, changes)
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

func addMemberIdsToChanges(client *gapi.Client, changes []MemberChange) ([]MemberChange, error) {
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
			if change.Type == AddMember {
				return nil, fmt.Errorf("error adding user %s. User does not exist in Grafana", change.Member.Email)
			} else {
				log.Printf("[DEBUG] Skipping removal of user %s. User does not exist in Grafana", change.Member.Email)
				continue
			}
		}

		change.Member.ID = id
		output = append(output, change)
	}
	return output, nil
}

func applyMemberChanges(client *gapi.Client, teamID int64, changes []MemberChange) diag.Diagnostics {
	var err error
	for _, change := range changes {
		u := change.Member
		switch change.Type {
		case AddMember:
			err = client.AddTeamMember(teamID, u.ID)
		case RemoveMember:
			err = client.RemoveMemberFromTeam(teamID, u.ID)
		}
		if err != nil {
			return diag.FromErr(err)
		}
	}
	return nil
}
