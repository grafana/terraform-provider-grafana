package grafana

import (
	"context"
	"fmt"
	"log"
	"strconv"

	gapi "github.com/grafana/grafana-api-golang-client"
	"github.com/grafana/terraform-provider-grafana/internal/common"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
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

		CreateContext: CreateTeam,
		ReadContext:   ReadTeam,
		UpdateContext: UpdateTeam,
		DeleteContext: DeleteTeam,
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
			"preferences": {
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"theme": {
							Type:         schema.TypeString,
							Optional:     true,
							ValidateFunc: validation.StringInSlice([]string{"light", "dark", ""}, false),
							Description:  "The default theme for this team. Available themes are `light`, `dark`, or an empty string for the default theme.",
							Default:      "",
						},
						"home_dashboard_uid": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "The UID of the dashboard to display when a team member logs in.",
							Default:     "",
						},
						"timezone": {
							Type:         schema.TypeString,
							Optional:     true,
							ValidateFunc: validation.StringInSlice([]string{"utc", "browser", ""}, false),
							Description:  "The default timezone for this team. Available values are `utc`, `browser`, or an empty string for the default.",
							Default:      "",
						},
					},
				},
			},
		},
	}
}

func CreateTeam(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*common.Client).GrafanaAPI
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

	if err := updateTeamPreferences(client, teamID, d); err != nil {
		return err
	}

	return ReadTeam(ctx, d, meta)
}

func ReadTeam(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	teamID, _ := strconv.ParseInt(d.Id(), 10, 64)
	return readTeamFromID(teamID, d, meta)
}

func readTeamFromID(teamID int64, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*common.Client).GrafanaAPI
	resp, err := client.Team(teamID)
	if err, shouldReturn := common.CheckReadError("team", d, err); shouldReturn {
		return err
	}

	d.SetId(strconv.FormatInt(teamID, 10))
	d.Set("team_id", teamID)
	d.Set("name", resp.Name)
	if resp.Email != "" {
		d.Set("email", resp.Email)
	}

	preferences, err := client.TeamPreferences(teamID)
	if err != nil {
		return diag.FromErr(err)
	}

	if preferences.Theme+preferences.Timezone+preferences.HomeDashboardUID != "" {
		d.Set("preferences", []map[string]interface{}{
			{
				"theme":              preferences.Theme,
				"home_dashboard_uid": preferences.HomeDashboardUID,
				"timezone":           preferences.Timezone,
			},
		})
	}

	return readTeamMembers(d, meta)
}

func UpdateTeam(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*common.Client).GrafanaAPI
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

	if err := updateTeamPreferences(client, teamID, d); err != nil {
		return err
	}

	return ReadTeam(ctx, d, meta)
}

func DeleteTeam(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*common.Client).GrafanaAPI
	teamID, _ := strconv.ParseInt(d.Id(), 10, 64)
	if err := client.DeleteTeam(teamID); err != nil {
		return diag.FromErr(err)
	}

	return diag.Diagnostics{}
}

func updateTeamPreferences(client *gapi.Client, teamID int64, d *schema.ResourceData) diag.Diagnostics {
	if d.IsNewResource() || d.HasChanges("preferences.0.theme", "preferences.0.home_dashboard_uid", "preferences.0.timezone") {
		preferences := gapi.Preferences{
			Theme:            d.Get("preferences.0.theme").(string),
			HomeDashboardUID: d.Get("preferences.0.home_dashboard_uid").(string),
			Timezone:         d.Get("preferences.0.timezone").(string),
		}

		return diag.FromErr(client.UpdateTeamPreferences(teamID, preferences))
	}

	return nil
}

func readTeamMembers(d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*common.Client).GrafanaAPI
	teamID, _ := strconv.ParseInt(d.Id(), 10, 64)
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
	client := meta.(*common.Client).GrafanaAPI
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

func applyMemberChanges(meta interface{}, teamID int64, changes []MemberChange) error {
	var err error
	client := meta.(*common.Client).GrafanaAPI
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
