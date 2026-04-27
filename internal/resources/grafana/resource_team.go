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
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
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

	// defaultIgnoreExternallySyncedMembers matches the schema Default for ignore_externally_synced_members.
	// Used in Read() to handle null state after SDKv2 → Framework migration.
	defaultIgnoreExternallySyncedMembers = true
)

var (
	_ resource.Resource                = &teamResource{}
	_ resource.ResourceWithConfigure   = &teamResource{}
	_ resource.ResourceWithImportState = &teamResource{}

	resourceTeamName = "grafana_team"
	resourceTeamID   = orgResourceIDInt("id")
)

// membersUseStateWhenUnconfigured returns a plan modifier for the members
// attribute that preserves the prior state value when the attribute is not
// set in config.
//
// This prevents accidental mass-removal of team members when a user manages a
// team without specifying the members attribute (e.g., members are managed by
// an external system like Okta team sync or SCIM).
//
// Behavior:
//   - Config sets members (even to []): use the config value (enforced by the framework).
//   - Config omits members, no prior state (create): default to empty set.
//   - Config omits members, has prior state (update): preserve prior state.
func membersUseStateWhenUnconfigured() planmodifier.Set {
	return &membersPreserveStatePlanModifier{}
}

type membersPreserveStatePlanModifier struct{}

func (m *membersPreserveStatePlanModifier) Description(_ context.Context) string {
	return "Preserves the prior state value when members is not configured, preventing accidental removal of externally managed team members."
}

func (m *membersPreserveStatePlanModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m *membersPreserveStatePlanModifier) PlanModifySet(_ context.Context, req planmodifier.SetRequest, resp *planmodifier.SetResponse) {
	// If the attribute is explicitly set in config, let the framework handle it.
	if !req.ConfigValue.IsNull() {
		return
	}

	// Config is null (attribute not in user's .tf file).
	if req.StateValue.IsNull() || req.StateValue.IsUnknown() {
		// Create path: no prior state. Default to empty set so a new team
		// starts with zero members (matching prior SDKv2 behavior).
		resp.PlanValue = types.SetValueMust(types.StringType, []attr.Value{})
		return
	}

	// Update path: prior state exists. Preserve it so that members managed
	// outside of Terraform (Okta, SCIM, team sync, UI) are not removed.
	resp.PlanValue = req.StateValue
}

func makeResourceTeam() *common.Resource {
	return common.NewResource(
		common.CategoryGrafanaOSS,
		resourceTeamName,
		resourceTeamID,
		&teamResource{},
	).
		WithLister(listerFunctionOrgResource(listTeams)).
		WithPreferredResourceNameField("name")
}

type resourceTeamPreferencesModel struct {
	Theme            types.String `tfsdk:"theme"`
	HomeDashboardUID types.String `tfsdk:"home_dashboard_uid"`
	Timezone         types.String `tfsdk:"timezone"`
	WeekStart        types.String `tfsdk:"week_start"`
}

type resourceTeamSyncModel struct {
	Groups types.Set `tfsdk:"groups"`
}

type resourceTeamModel struct {
	ID                            types.String                   `tfsdk:"id"`
	OrgID                         types.String                   `tfsdk:"org_id"`
	TeamID                        types.Int64                    `tfsdk:"team_id"`
	TeamUID                       types.String                   `tfsdk:"team_uid"`
	Name                          types.String                   `tfsdk:"name"`
	Email                         types.String                   `tfsdk:"email"`
	Members                       types.Set                      `tfsdk:"members"`
	IgnoreExternallySyncedMembers types.Bool                     `tfsdk:"ignore_externally_synced_members"`
	Preferences                   []resourceTeamPreferencesModel `tfsdk:"preferences"`
	TeamSync                      []resourceTeamSyncModel        `tfsdk:"team_sync"`
}

type teamResource struct {
	basePluginFrameworkResource
}

func (r *teamResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = resourceTeamName
}

func (r *teamResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `
* [Official documentation](https://grafana.com/docs/grafana/latest/administration/team-management/)
* [HTTP API](https://grafana.com/docs/grafana/latest/developer-resources/api-reference/http-api/api-legacy/team/)
`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"org_id": pluginFrameworkOrgIDAttribute(),
			"team_id": schema.Int64Attribute{
				Computed:    true,
				Description: "The team id assigned to this team by Grafana.",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"team_uid": schema.StringAttribute{
				Computed:    true,
				Description: "The team uid assigned to this team by Grafana.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The display name for the Grafana team created.",
			},
			"email": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "An email address for the team.",
			},
			"members": schema.SetAttribute{
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Description: "A set of email addresses corresponding to users who should be given membership to the team. Note: users specified here must already exist in Grafana.",
				PlanModifiers: []planmodifier.Set{
					membersUseStateWhenUnconfigured(),
				},
			},
			"ignore_externally_synced_members": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(true),
				Description: "Ignores team members that have been added to team by " +
					"[Team Sync](https://grafana.com/docs/grafana/latest/setup-grafana/configure-security/configure-team-sync/). " +
					"Team Sync can be provisioned using [grafana_team_external_group resource](https://registry.terraform.io/providers/grafana/grafana/latest/docs/resources/team_external_group).",
			},
		},
		// preferences and team_sync use Blocks (not Attributes) for protocol v5 mux compatibility.
		Blocks: map[string]schema.Block{
			"preferences": schema.ListNestedBlock{
				Validators: []validator.List{
					listvalidator.SizeAtMost(1),
				},
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"theme": schema.StringAttribute{
							Optional:    true,
							Computed:    true,
							Default:     stringdefault.StaticString(""),
							Description: "The default theme for this team. Available themes are `light`, `dark`, `system`, or an empty string for the default theme.",
							Validators:  []validator.String{stringvalidator.OneOf("light", "dark", "system", "")},
						},
						"home_dashboard_uid": schema.StringAttribute{
							Optional:    true,
							Computed:    true,
							Default:     stringdefault.StaticString(""),
							Description: "The UID of the dashboard to display when a team member logs in.",
						},
						"timezone": schema.StringAttribute{
							Optional:    true,
							Computed:    true,
							Default:     stringdefault.StaticString(""),
							Description: "The default timezone for this team. Available values are `utc`, `browser`, or an empty string for the default.",
							Validators:  []validator.String{stringvalidator.OneOf("utc", "browser", "")},
						},
						"week_start": schema.StringAttribute{
							Optional:    true,
							Computed:    true,
							Default:     stringdefault.StaticString(""),
							Description: "The default week start day for this team. Available values are `sunday`, `monday`, `saturday`, or an empty string for the default.",
							Validators:  []validator.String{stringvalidator.OneOf("sunday", "monday", "saturday", "")},
						},
					},
				},
			},
			"team_sync": schema.ListNestedBlock{
				MarkdownDescription: "Sync external auth provider groups with this Grafana team. Only available in Grafana Enterprise.\n" +
					"* [Official documentation](https://grafana.com/docs/grafana/latest/setup-grafana/configure-security/configure-team-sync/)\n" +
					"* [HTTP API](https://grafana.com/docs/grafana/latest/developer-resources/api-reference/http-api/api-legacy/team_sync/)",
				Validators: []validator.List{
					listvalidator.SizeAtMost(1),
				},
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"groups": schema.SetAttribute{
							Optional:    true,
							ElementType: types.StringType,
						},
					},
				},
			},
		},
	}
}

func (r *teamResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data resourceTeamModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if orgIDStr := data.OrgID.ValueString(); orgIDStr != "" && orgIDStr != "0" && r.config.APIKey != "" {
		resp.Diagnostics.AddError("Invalid configuration", "org_id is only supported with basic auth. API keys are already org-scoped")
		return
	}

	client, orgID, err := r.clientFromNewOrgResource(data.OrgID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to get client", err.Error())
		return
	}

	createResp, err := client.Teams.CreateTeam(&models.CreateTeamCommand{
		Name:  data.Name.ValueString(),
		Email: data.Email.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create team", err.Error())
		return
	}
	teamID := createResp.GetPayload().TeamID
	teamIDStr := strconv.FormatInt(teamID, 10)

	data.ID = types.StringValue(MakeOrgResourceID(orgID, teamID))
	data.TeamID = types.Int64Value(teamID)

	// Apply members — Members may be unknown on first create (Optional+Computed, no prior state).
	var planMembers []string
	if !data.Members.IsNull() && !data.Members.IsUnknown() {
		resp.Diagnostics.Append(data.Members.ElementsAs(ctx, &planMembers, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}
	if err := applyTeamMembers(client, teamID, nil, planMembers); err != nil {
		resp.Diagnostics.AddError("Failed to update team members", err.Error())
		return
	}

	// Apply preferences
	if len(data.Preferences) > 0 {
		p := data.Preferences[0]
		if _, err := client.Teams.UpdateTeamPreferences(teamIDStr, &models.UpdatePrefsCmd{
			Theme:            p.Theme.ValueString(),
			HomeDashboardUID: p.HomeDashboardUID.ValueString(),
			Timezone:         p.Timezone.ValueString(),
			WeekStart:        p.WeekStart.ValueString(),
		}); err != nil {
			resp.Diagnostics.AddError("Failed to update team preferences", err.Error())
			return
		}
	}

	// Apply team sync groups
	if len(data.TeamSync) > 0 {
		var planGroups []string
		if !data.TeamSync[0].Groups.IsNull() && !data.TeamSync[0].Groups.IsUnknown() {
			resp.Diagnostics.Append(data.TeamSync[0].Groups.ElementsAs(ctx, &planGroups, false)...)
			if resp.Diagnostics.HasError() {
				return
			}
		}
		if err := applyTeamExternalGroup(client, teamID, planGroups, nil); err != nil {
			resp.Diagnostics.AddError("Failed to update team sync groups", err.Error())
			return
		}
	}

	readData, diags := r.read(ctx, data.ID.ValueString(), data.IgnoreExternallySyncedMembers.ValueBool(), len(data.TeamSync) > 0)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, readData)...)
}

func (r *teamResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data resourceTeamModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Default to true when state is null (e.g. after SDKv2 → Framework migration
	// or for resources created before this attribute existed). This matches the
	// behavior of the old SDKv2 DiffSuppressFunc which suppressed diffs when
	// old="" and new="true".
	ignoreExternallySynced := data.IgnoreExternallySyncedMembers.ValueBool()
	if data.IgnoreExternallySyncedMembers.IsNull() || data.IgnoreExternallySyncedMembers.IsUnknown() {
		ignoreExternallySynced = defaultIgnoreExternallySyncedMembers
	}

	readData, diags := r.read(ctx, data.ID.ValueString(), ignoreExternallySynced, len(data.TeamSync) > 0)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if readData == nil {
		resp.State.RemoveResource(ctx)
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, readData)...)
}

func (r *teamResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var planData resourceTeamModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &planData)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var stateData resourceTeamModel
	resp.Diagnostics.Append(req.State.Get(ctx, &stateData)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var configData resourceTeamModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &configData)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client, _, split, err := r.clientFromExistingOrgResource(resourceTeamID, planData.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to parse resource ID", err.Error())
		return
	}
	teamID := split[0].(int64)
	teamIDStr := strconv.FormatInt(teamID, 10)

	// Update name/email
	if _, err := client.Teams.UpdateTeam(teamIDStr, &models.UpdateTeamCommand{
		Name:  planData.Name.ValueString(),
		Email: planData.Email.ValueString(),
	}); err != nil {
		resp.Diagnostics.AddError("Failed to update team", err.Error())
		return
	}

	// Update members: diff state vs plan
	var stateMembers, planMembers []string
	if !stateData.Members.IsNull() && !stateData.Members.IsUnknown() {
		resp.Diagnostics.Append(stateData.Members.ElementsAs(ctx, &stateMembers, false)...)
	}
	if !planData.Members.IsNull() && !planData.Members.IsUnknown() {
		resp.Diagnostics.Append(planData.Members.ElementsAs(ctx, &planMembers, false)...)
	}
	if resp.Diagnostics.HasError() {
		return
	}
	if err := applyTeamMembers(client, teamID, stateMembers, planMembers); err != nil {
		resp.Diagnostics.AddError("Failed to update team members", err.Error())
		return
	}

	// Update preferences
	if len(planData.Preferences) > 0 {
		p := planData.Preferences[0]
		if _, err := client.Teams.UpdateTeamPreferences(teamIDStr, &models.UpdatePrefsCmd{
			Theme:            p.Theme.ValueString(),
			HomeDashboardUID: p.HomeDashboardUID.ValueString(),
			Timezone:         p.Timezone.ValueString(),
			WeekStart:        p.WeekStart.ValueString(),
		}); err != nil {
			resp.Diagnostics.AddError("Failed to update team preferences", err.Error())
			return
		}
	} else if len(stateData.Preferences) > 0 {
		// Preferences block was removed; reset to defaults.
		if _, err := client.Teams.UpdateTeamPreferences(teamIDStr, &models.UpdatePrefsCmd{}); err != nil {
			resp.Diagnostics.AddError("Failed to reset team preferences", err.Error())
			return
		}
	}

	// Update team sync: diff state vs plan groups
	var stateGroups, planGroups []string
	if len(stateData.TeamSync) > 0 && !stateData.TeamSync[0].Groups.IsNull() && !stateData.TeamSync[0].Groups.IsUnknown() {
		resp.Diagnostics.Append(stateData.TeamSync[0].Groups.ElementsAs(ctx, &stateGroups, false)...)
	}
	if len(planData.TeamSync) > 0 && !planData.TeamSync[0].Groups.IsNull() && !planData.TeamSync[0].Groups.IsUnknown() {
		resp.Diagnostics.Append(planData.TeamSync[0].Groups.ElementsAs(ctx, &planGroups, false)...)
	}
	if resp.Diagnostics.HasError() {
		return
	}
	if len(planData.TeamSync) > 0 || len(stateData.TeamSync) > 0 {
		add, remove := teamSyncGroupDiff(stateGroups, planGroups)
		if err := applyTeamExternalGroup(client, teamID, add, remove); err != nil {
			resp.Diagnostics.AddError("Failed to update team sync groups", err.Error())
			return
		}
	}

	// When members is not in config, the plan modifier preserved state members
	// (which were read with the old ignore value). Use the same ignore value
	// for the final read so the returned member list matches the plan.
	// On the next Read() (refresh), the new ignore value will take effect and
	// silently update the member list in state.
	readIgnore := planData.IgnoreExternallySyncedMembers.ValueBool()
	if configData.Members.IsNull() {
		readIgnore = stateData.IgnoreExternallySyncedMembers.ValueBool()
		if stateData.IgnoreExternallySyncedMembers.IsNull() || stateData.IgnoreExternallySyncedMembers.IsUnknown() {
			readIgnore = defaultIgnoreExternallySyncedMembers
		}
	}

	readData, diags := r.read(ctx, planData.ID.ValueString(), readIgnore, len(planData.TeamSync) > 0)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	// Override ignore to the plan value so state reflects the desired config.
	readData.IgnoreExternallySyncedMembers = planData.IgnoreExternallySyncedMembers
	resp.Diagnostics.Append(resp.State.Set(ctx, readData)...)
}

func (r *teamResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data resourceTeamModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client, _, split, err := r.clientFromExistingOrgResource(resourceTeamID, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to parse resource ID", err.Error())
		return
	}
	teamIDStr := strconv.FormatInt(split[0].(int64), 10)

	_, err = client.Teams.DeleteTeamByID(teamIDStr)
	if err != nil && !common.IsNotFoundError(err) {
		resp.Diagnostics.AddError("Failed to delete team", err.Error())
	}
}

func (r *teamResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import without reading team sync (Enterprise-only; safe to omit for OSS import).
	readData, diags := r.read(ctx, req.ID, true, false)
	resp.Diagnostics = diags
	if resp.Diagnostics.HasError() {
		return
	}
	if readData == nil {
		resp.Diagnostics.AddError("Resource not found", "Team not found during import")
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, readData)...)
}

func (r *teamResource) read(ctx context.Context, id string, ignoreExternallySynced bool, readTeamSync bool) (*resourceTeamModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	client, _, split, err := r.clientFromExistingOrgResource(resourceTeamID, id)
	if err != nil {
		diags.AddError("Failed to parse resource ID", err.Error())
		return nil, diags
	}
	teamID := split[0].(int64)
	teamIDStr := strconv.FormatInt(teamID, 10)

	team, err := getTeamByID(client, teamID)
	if err != nil {
		if common.IsNotFoundError(err) {
			return nil, diags
		}
		diags.AddError("Failed to read team", err.Error())
		return nil, diags
	}

	emailVal := types.StringValue(team.Email)

	data := &resourceTeamModel{
		ID:                            types.StringValue(MakeOrgResourceID(team.OrgID, teamID)),
		OrgID:                         types.StringValue(strconv.FormatInt(team.OrgID, 10)),
		TeamID:                        types.Int64Value(teamID),
		TeamUID:                       types.StringValue(team.UID),
		Name:                          types.StringValue(team.Name),
		Email:                         emailVal,
		IgnoreExternallySyncedMembers: types.BoolValue(ignoreExternallySynced),
	}

	// Preferences
	prefsResp, err := client.Teams.GetTeamPreferences(teamIDStr)
	if err != nil {
		diags.AddError("Failed to read team preferences", err.Error())
		return nil, diags
	}
	prefs := prefsResp.GetPayload()
	if prefs.Theme != "" || prefs.Timezone != "" || prefs.HomeDashboardUID != "" || prefs.WeekStart != "" {
		data.Preferences = []resourceTeamPreferencesModel{{
			Theme:            types.StringValue(prefs.Theme),
			HomeDashboardUID: types.StringValue(prefs.HomeDashboardUID),
			Timezone:         types.StringValue(prefs.Timezone),
			WeekStart:        types.StringValue(prefs.WeekStart),
		}}
	}

	// Team sync (Enterprise-only; caller controls whether to attempt)
	if readTeamSync {
		syncResp, err := client.SyncTeamGroups.GetTeamGroupsAPI(teamID)
		if err != nil {
			diags.AddError("Failed to read team sync groups", err.Error())
			return nil, diags
		}
		groupStrs := make([]string, 0, len(syncResp.GetPayload()))
		for _, g := range syncResp.GetPayload() {
			groupStrs = append(groupStrs, g.GroupID)
		}
		groupSet, setDiags := types.SetValueFrom(ctx, types.StringType, groupStrs)
		diags.Append(setDiags...)
		if diags.HasError() {
			return nil, diags
		}
		data.TeamSync = []resourceTeamSyncModel{{Groups: groupSet}}
	}

	// Members
	membersResp, err := client.Teams.GetTeamMembers(teamIDStr)
	if err != nil {
		diags.AddError("Failed to read team members", err.Error())
		return nil, diags
	}
	memberSlice := []string{}
	for _, m := range membersResp.GetPayload() {
		if m.Email == "admin@localhost" {
			continue
		}
		if ignoreExternallySynced && len(m.Labels) > 0 {
			continue
		}
		memberSlice = append(memberSlice, m.Email)
	}
	memberSet, setDiags := types.SetValueFrom(ctx, types.StringType, memberSlice)
	diags.Append(setDiags...)
	if diags.HasError() {
		return nil, diags
	}
	data.Members = memberSet

	return data, diags
}

// applyTeamMembers computes and applies member additions/removals.
func applyTeamMembers(client *goapi.GrafanaHTTPAPI, teamID int64, stateEmails, planEmails []string) error {
	stateMap := make(map[string]TeamMember, len(stateEmails))
	for _, email := range stateEmails {
		stateMap[email] = TeamMember{0, email}
	}
	planMap := make(map[string]TeamMember, len(planEmails))
	for _, email := range planEmails {
		planMap[email] = TeamMember{0, email}
	}
	changes := memberChanges(stateMap, planMap)
	changes, err := addMemberIdsToChanges(client, changes)
	if err != nil {
		return err
	}
	return applyMemberChanges(client, teamID, changes)
}

// teamSyncGroupDiff returns which groups to add and which to remove.
func teamSyncGroupDiff(current, desired []string) (add, remove []string) {
	currentSet := make(map[string]bool, len(current))
	for _, g := range current {
		currentSet[g] = true
	}
	desiredSet := make(map[string]bool, len(desired))
	for _, g := range desired {
		desiredSet[g] = true
	}
	for _, g := range desired {
		if !currentSet[g] {
			add = append(add, g)
		}
	}
	for _, g := range current {
		if !desiredSet[g] {
			remove = append(remove, g)
		}
	}
	return
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

func memberChanges(stateMembers, configMembers map[string]TeamMember) []MemberChange {
	var changes []MemberChange
	for _, user := range configMembers {
		if _, ok := stateMembers[user.Email]; !ok {
			changes = append(changes, MemberChange{AddMember, user})
		}
	}
	for _, user := range stateMembers {
		if _, ok := configMembers[user.Email]; !ok {
			changes = append(changes, MemberChange{RemoveMember, user})
		}
	}
	return changes
}

func addMemberIdsToChanges(client *goapi.GrafanaHTTPAPI, changes []MemberChange) ([]MemberChange, error) {
	resp, err := client.Org.GetOrgUsersForCurrentOrg(nil)
	if err != nil {
		return nil, err
	}
	gUserMap := make(map[string]int64, len(resp.GetPayload()))
	for _, u := range resp.GetPayload() {
		gUserMap[u.Email] = u.UserID
	}

	var output []MemberChange
	for _, change := range changes {
		id, ok := gUserMap[change.Member.Email]
		if !ok {
			if change.Type == AddMember {
				return nil, fmt.Errorf("error adding user %s. User does not exist in Grafana", change.Member.Email)
			}
			log.Printf("[DEBUG] Skipping removal of user %s. User does not exist in Grafana", change.Member.Email)
			continue
		}
		change.Member.ID = id
		output = append(output, change)
	}
	return output, nil
}

func applyMemberChanges(client *goapi.GrafanaHTTPAPI, teamID int64, changes []MemberChange) error {
	for _, change := range changes {
		u := change.Member
		var err error
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
