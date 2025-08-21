package grafana

import (
	"context"
	"fmt"
	"log"
	"strconv"

	goapi "github.com/grafana/grafana-openapi-client-go/client"
	"github.com/grafana/grafana-openapi-client-go/client/teams"
	"github.com/grafana/grafana-openapi-client-go/models"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
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

func resourceTeam() *common.Resource {
	schema := &schema.Resource{

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
			"org_id": orgIDAttribute(),
			"team_id": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "The team id assigned to this team by Grafana.",
			},
			"team_uid": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The team uid assigned to this team by Grafana.",
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
				DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
					return old == new || (old == "" && new == "true")
				},
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
							ValidateFunc: validation.StringInSlice([]string{"light", "dark", "system", ""}, false),
							Description:  "The default theme for this team. Available themes are `light`, `dark`, `system`, or an empty string for the default theme.",
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
						"week_start": {
							Type:         schema.TypeString,
							Optional:     true,
							ValidateFunc: validation.StringInSlice([]string{"sunday", "monday", "saturday", ""}, false),
							Description:  "The default week start day for this team. Available values are `sunday`, `monday`, `saturday`, or an empty string for the default.",
							Default:      "",
						},
					},
				},
			},
			"team_sync": {
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"groups": {
							Type:     schema.TypeSet,
							Optional: true,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},
					},
				},
				Description: `Sync external auth provider groups with this Grafana team. Only available in Grafana Enterprise.
	* [Official documentation](https://grafana.com/docs/grafana/latest/setup-grafana/configure-security/configure-team-sync/)
	* [HTTP API](https://grafana.com/docs/grafana/latest/developers/http_api/team_sync/)
`,
			},
		},
	}

	return common.NewLegacySDKResource(
		common.CategoryGrafanaOSS,
		"grafana_team",
		orgResourceIDInt("id"),
		schema,
	).
		WithLister(listerFunctionOrgResource(listTeams)).
		WithPreferredResourceNameField("name")
}

func listTeams(ctx context.Context, client *goapi.GrafanaHTTPAPI, orgID int64) ([]string, error) {
	var ids []string
	var page int64 = 1
	for {
		params := teams.NewSearchTeamsParams().WithPage(&page)
		resp, err := client.Teams.SearchTeams(params)
		if err != nil {
			return nil, err
		}

		for _, team := range resp.Payload.Teams {
			ids = append(ids, MakeOrgResourceID(orgID, team.ID))
		}

		if resp.Payload.TotalCount <= int64(len(ids)) {
			break
		}

		page++
	}

	return ids, nil
}

func CreateTeam(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, orgID := OAPIClientFromNewOrgResource(meta, d)
	body := models.CreateTeamCommand{
		Name:  d.Get("name").(string),
		Email: d.Get("email").(string),
	}
	resp, err := client.Teams.CreateTeam(&body)
	if err != nil {
		return diag.Errorf("error creating team: %s", err)
	}
	teamID := resp.GetPayload().TeamID

	d.SetId(MakeOrgResourceID(orgID, teamID))
	d.Set("team_id", teamID)
	if err = UpdateMembers(client, d); err != nil {
		return diag.FromErr(err)
	}

	if err := updateTeamPreferences(client, teamID, d); err != nil {
		return err
	}

	if _, ok := d.GetOk("team_sync"); ok {
		if err := manageTeamExternalGroup(client, teamID, d, "team_sync.0.groups"); err != nil {
			return diag.FromErr(err)
		}
	}

	return ReadTeam(ctx, d, meta)
}

func ReadTeam(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, _, idStr := OAPIClientFromExistingOrgResource(meta, d.Id())
	teamID, _ := strconv.ParseInt(idStr, 10, 64)
	_, readTeamSync := d.GetOk("team_sync")
	return readTeamFromID(client, teamID, d, readTeamSync)
}

func readTeamFromID(client *goapi.GrafanaHTTPAPI, teamID int64, d *schema.ResourceData, readTeamSync bool) diag.Diagnostics {
	teamIDStr := strconv.FormatInt(teamID, 10)
	team, err := getTeamByID(client, teamID)
	if err, shouldReturn := common.CheckReadError("team", d, err); shouldReturn {
		return err
	}

	d.SetId(MakeOrgResourceID(team.OrgID, teamID))
	d.Set("team_id", teamID)
	d.Set("team_uid", team.UID)
	d.Set("name", team.Name)
	d.Set("org_id", strconv.FormatInt(team.OrgID, 10))
	if team.Email != "" {
		d.Set("email", team.Email)
	}

	resp, err := client.Teams.GetTeamPreferences(teamIDStr)
	if err != nil {
		return diag.FromErr(err)
	}
	preferences := resp.GetPayload()

	if readTeamSync {
		resp, err := client.SyncTeamGroups.GetTeamGroupsAPI(teamID)
		if err != nil {
			return diag.FromErr(err)
		}
		teamGroups := resp.GetPayload()

		groupIDs := make([]string, 0, len(teamGroups))
		for _, teamGroup := range teamGroups {
			groupIDs = append(groupIDs, teamGroup.GroupID)
		}
		d.Set("team_sync", []map[string]interface{}{
			{
				"groups": groupIDs,
			},
		})
	}

	if preferences.Theme+preferences.Timezone+preferences.HomeDashboardUID+preferences.WeekStart != "" {
		d.Set("preferences", []map[string]interface{}{
			{
				"theme":              preferences.Theme,
				"home_dashboard_uid": preferences.HomeDashboardUID,
				"timezone":           preferences.Timezone,
				"week_start":         preferences.WeekStart,
			},
		})
	}

	return readTeamMembers(client, d)
}

func UpdateTeam(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, _, idStr := OAPIClientFromExistingOrgResource(meta, d.Id())
	teamID, _ := strconv.ParseInt(idStr, 10, 64)
	if d.HasChange("name") || d.HasChange("email") {
		name := d.Get("name").(string)
		email := d.Get("email").(string)
		body := models.UpdateTeamCommand{
			Name:  name,
			Email: email,
		}
		if _, err := client.Teams.UpdateTeam(idStr, &body); err != nil {
			return diag.FromErr(err)
		}
	}
	if err := UpdateMembers(client, d); err != nil {
		return diag.FromErr(err)
	}

	if err := updateTeamPreferences(client, teamID, d); err != nil {
		return err
	}

	if _, ok := d.GetOk("team_sync"); ok {
		if err := manageTeamExternalGroup(client, teamID, d, "team_sync.0.groups"); err != nil {
			return diag.FromErr(err)
		}
	}

	return ReadTeam(ctx, d, meta)
}

func DeleteTeam(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, _, idStr := OAPIClientFromExistingOrgResource(meta, d.Id())
	_, err := client.Teams.DeleteTeamByID(idStr)
	diag, _ := common.CheckReadError("team", d, err)
	return diag
}

func updateTeamPreferences(client *goapi.GrafanaHTTPAPI, teamID int64, d *schema.ResourceData) diag.Diagnostics {
	if d.IsNewResource() || d.HasChanges("preferences.0.theme", "preferences.0.home_dashboard_uid", "preferences.0.timezone", "preferences.0.week_start") {
		body := models.UpdatePrefsCmd{
			Theme:            d.Get("preferences.0.theme").(string),
			HomeDashboardUID: d.Get("preferences.0.home_dashboard_uid").(string),
			Timezone:         d.Get("preferences.0.timezone").(string),
			WeekStart:        d.Get("preferences.0.week_start").(string),
		}
		_, err := client.Teams.UpdateTeamPreferences(strconv.FormatInt(teamID, 10), &body)
		return diag.FromErr(err)
	}

	return nil
}

func readTeamMembers(client *goapi.GrafanaHTTPAPI, d *schema.ResourceData) diag.Diagnostics {
	resp, err := client.Teams.GetTeamMembers(strconv.Itoa(d.Get("team_id").(int)))
	if err != nil {
		return diag.FromErr(err)
	}
	teamMembers := resp.GetPayload()
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

func UpdateMembers(client *goapi.GrafanaHTTPAPI, d *schema.ResourceData) error {
	stateMembers, configMembers, err := collectMembers(d)
	if err != nil {
		return err
	}
	// compile the list of differences between current state and config
	changes := memberChanges(stateMembers, configMembers)
	// retrieves the corresponding user IDs based on the email provided
	changes, err = addMemberIdsToChanges(client, changes)
	if err != nil {
		return err
	}
	// now we can make the corresponding updates so current state matches config
	return applyMemberChanges(client, int64(d.Get("team_id").(int)), changes)
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

func addMemberIdsToChanges(client *goapi.GrafanaHTTPAPI, changes []MemberChange) ([]MemberChange, error) {
	gUserMap := make(map[string]int64)

	resp, err := client.Org.GetOrgUsersForCurrentOrg(nil)
	if err != nil {
		return nil, err
	}
	gUsers := resp.GetPayload()
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

func applyMemberChanges(client *goapi.GrafanaHTTPAPI, teamID int64, changes []MemberChange) error {
	var err error
	for _, change := range changes {
		u := change.Member
		switch change.Type {
		case AddMember:
			_, err = client.Teams.AddTeamMember(strconv.FormatInt(teamID, 10), &models.AddTeamMemberCommand{UserID: u.ID})
		case RemoveMember:
			_, err = client.Teams.RemoveTeamMember(u.ID, strconv.FormatInt(teamID, 10))
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func getTeamByID(client *goapi.GrafanaHTTPAPI, teamID int64) (*models.TeamDTO, error) {
	resp, err := client.Teams.GetTeamByID(strconv.FormatInt(teamID, 10))
	if err != nil {
		return nil, err
	}
	return resp.GetPayload(), nil
}
