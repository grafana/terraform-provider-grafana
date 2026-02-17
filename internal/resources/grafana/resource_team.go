package grafana

import (
	"context"
	"fmt"
	"log"
	"slices"
	"strconv"

	goapi "github.com/grafana/grafana-openapi-client-go/client"
	"github.com/grafana/grafana-openapi-client-go/client/teams"
	"github.com/grafana/grafana-openapi-client-go/models"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	sdkschema "github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	sdkdiag "github.com/hashicorp/terraform-plugin-sdk/v2/diag"
)

const resourceTeamName = "grafana_team"

var resourceTeamID = orgResourceIDInt("id")

var _ resource.Resource = (*teamResource)(nil)
var _ resource.ResourceWithImportState = (*teamResource)(nil)

func resourceTeam() *common.Resource {
	return common.NewResource(
		common.CategoryGrafanaOSS,
		resourceTeamName,
		resourceTeamID,
		&teamResource{},
	).
		WithLister(listerFunctionOrgResource(listTeams)).
		WithPreferredResourceNameField("name")
}

// legacyTeamSchema returns the SDKv2 schema for the team resource. Used by the grafana_team data source to clone attributes.
func legacyTeamSchema() *sdkschema.Resource {
	return &sdkschema.Resource{
		Schema: map[string]*sdkschema.Schema{
			"org_id":     orgIDAttribute(),
			"team_id":   {Type: sdkschema.TypeInt, Computed: true, Description: "The team id assigned to this team by Grafana."},
			"team_uid":  {Type: sdkschema.TypeString, Computed: true, Description: "The team uid assigned to this team by Grafana."},
			"name":      {Type: sdkschema.TypeString, Required: true, Description: "The display name for the Grafana team created."},
			"email":     {Type: sdkschema.TypeString, Optional: true, Description: "An email address for the team."},
			"members":   {Type: sdkschema.TypeSet, Optional: true, Elem: &sdkschema.Schema{Type: sdkschema.TypeString}, Description: "A set of email addresses corresponding to users who should be given membership to the team."},
			"ignore_externally_synced_members": {Type: sdkschema.TypeBool, Optional: true, Default: true, Description: "Ignores team members that have been added to team by Team Sync."},
			"preferences": {
				Type: sdkschema.TypeList, Optional: true, MaxItems: 1,
				Elem: &sdkschema.Resource{
					Schema: map[string]*sdkschema.Schema{
						"theme":               {Type: sdkschema.TypeString, Optional: true, Default: ""},
						"home_dashboard_uid": {Type: sdkschema.TypeString, Optional: true, Default: ""},
						"timezone":            {Type: sdkschema.TypeString, Optional: true, Default: ""},
						"week_start":          {Type: sdkschema.TypeString, Optional: true, Default: ""},
					},
				},
			},
			"team_sync": {
				Type: sdkschema.TypeList, Optional: true, MaxItems: 1,
				Elem: &sdkschema.Resource{
					Schema: map[string]*sdkschema.Schema{
						"groups": {Type: sdkschema.TypeSet, Optional: true, Elem: &sdkschema.Schema{Type: sdkschema.TypeString}},
					},
				},
			},
		},
	}
}

type teamResourceModel struct {
	ID                          types.String `tfsdk:"id"`
	OrgID                       types.String `tfsdk:"org_id"`
	TeamID                      types.Int64  `tfsdk:"team_id"`
	TeamUID                     types.String `tfsdk:"team_uid"`
	Name                        types.String `tfsdk:"name"`
	Email                       types.String `tfsdk:"email"`
	Members                     types.Set    `tfsdk:"members"`
	IgnoreExternallySyncedMembers types.Bool   `tfsdk:"ignore_externally_synced_members"`
	Preferences                 types.List   `tfsdk:"preferences"`
	TeamSync                    types.List   `tfsdk:"team_sync"`
}

type teamResource struct {
	basePluginFrameworkResource
}

func (r *teamResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = resourceTeamName
}

func (r *teamResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `
* [Official documentation](https://grafana.com/docs/grafana/latest/administration/team-management/)
* [HTTP API](https://grafana.com/docs/grafana/latest/developers/http_api/team/)
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
				Description: "An email address for the team.",
			},
			"members": schema.SetAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Description: `
A set of email addresses corresponding to users who should be given membership
to the team. Note: users specified here must already exist in Grafana.
`,
			},
			"ignore_externally_synced_members": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(true),
				Description: `
Ignores team members that have been added to team by [Team Sync](https://grafana.com/docs/grafana/latest/setup-grafana/configure-security/configure-team-sync/).
Team Sync can be provisioned using [grafana_team_external_group resource](https://registry.terraform.io/providers/grafana/grafana/latest/docs/resources/team_external_group).
`,
			},
			"preferences": schema.ListNestedAttribute{
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"theme": schema.StringAttribute{
							Optional: true,
							Computed: true,
							Default:  stringdefault.StaticString(""),
							Description: "The default theme for this team. Available themes are `light`, `dark`, `system`, or an empty string for the default theme.",
							Validators: []validator.String{
								stringvalidator.OneOf("light", "dark", "system", ""),
							},
						},
						"home_dashboard_uid": schema.StringAttribute{
							Optional:    true,
							Description: "The UID of the dashboard to display when a team member logs in.",
						},
						"timezone": schema.StringAttribute{
							Optional: true,
							Computed: true,
							Default:  stringdefault.StaticString(""),
							Description: "The default timezone for this team. Available values are `utc`, `browser`, or an empty string for the default.",
							Validators: []validator.String{
								stringvalidator.OneOf("utc", "browser", ""),
							},
						},
						"week_start": schema.StringAttribute{
							Optional: true,
							Computed: true,
							Default:  stringdefault.StaticString(""),
							Description: "The default week start day for this team. Available values are `sunday`, `monday`, `saturday`, or an empty string for the default.",
							Validators: []validator.String{
								stringvalidator.OneOf("sunday", "monday", "saturday", ""),
							},
						},
					},
				},
				Optional:    true,
				Description: "Team preferences.",
			},
			"team_sync": schema.ListNestedAttribute{
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"groups": schema.SetAttribute{
							ElementType: types.StringType,
							Optional:    true,
						},
					},
				},
				Optional: true,
				Description: `Sync external auth provider groups with this Grafana team. Only available in Grafana Enterprise.
* [Official documentation](https://grafana.com/docs/grafana/latest/setup-grafana/configure-security/configure-team-sync/)
* [HTTP API](https://grafana.com/docs/grafana/latest/developers/http_api/team_sync/)
`,
			},
		},
	}
}

func (r *teamResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan teamResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client, orgID, err := r.clientFromNewOrgResource(plan.OrgID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to get client", err.Error())
		return
	}

	body := models.CreateTeamCommand{
		Name:  plan.Name.ValueString(),
		Email: plan.Email.ValueString(),
	}
	createResp, err := client.Teams.CreateTeam(&body)
	if err != nil {
		resp.Diagnostics.AddError("Error creating team", err.Error())
		return
	}
	teamID := createResp.GetPayload().TeamID

	// Update members (config vs empty state)
	planMemberEmails := setToStringSlice(plan.Members)
	if err := updateTeamMembersFromSlices(client, teamID, nil, planMemberEmails); err != nil {
		resp.Diagnostics.AddError("Error updating team members", err.Error())
		return
	}

	// Preferences
	theme, homeUID, timezone, weekStart := teamPreferencesFromModel(plan.Preferences)
	if err := updateTeamPreferencesFromValues(client, teamID, theme, homeUID, timezone, weekStart); err != nil {
		resp.Diagnostics.AddError("Error updating team preferences", err.Error())
		return
	}

	// Team sync
	if !plan.TeamSync.IsNull() && len(plan.TeamSync.Elements()) > 0 {
		planGroups := teamSyncGroupsFromModel(plan.TeamSync)
		if err := applyTeamExternalGroup(client, teamID, planGroups, nil); err != nil {
			resp.Diagnostics.AddError("Error configuring team sync", err.Error())
			return
		}
	}

	// Read back into state
	data := plan
	data.ID = types.StringValue(MakeOrgResourceID(orgID, teamID))
	data.OrgID = types.StringValue(strconv.FormatInt(orgID, 10))
	data.TeamID = types.Int64Value(teamID)

	readData, diags := r.readTeamFromID(ctx, client, teamID, &data)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, readData)...)
}

func (r *teamResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state teamResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client, _, idFields, err := r.clientFromExistingOrgResource(resourceTeamID, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to get client", err.Error())
		return
	}
	teamID := idFields[0].(int64)

	readData, diags := r.readTeamFromID(ctx, client, teamID, &state)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}
	if readData == nil {
		resp.State.RemoveResource(ctx)
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, readData)...)
}

func (r *teamResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state teamResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client, _, idFields, err := r.clientFromExistingOrgResource(resourceTeamID, plan.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to get client", err.Error())
		return
	}
	teamID := idFields[0].(int64)
	teamIDStr := strconv.FormatInt(teamID, 10)

	if !plan.Name.Equal(state.Name) || !plan.Email.Equal(state.Email) {
		body := models.UpdateTeamCommand{
			Name:  plan.Name.ValueString(),
			Email: plan.Email.ValueString(),
		}
		if _, err := client.Teams.UpdateTeam(teamIDStr, &body); err != nil {
			resp.Diagnostics.AddError("Error updating team", err.Error())
			return
		}
	}

	stateMemberEmails := setToStringSlice(state.Members)
	planMemberEmails := setToStringSlice(plan.Members)
	if err := updateTeamMembersFromSlices(client, teamID, stateMemberEmails, planMemberEmails); err != nil {
		resp.Diagnostics.AddError("Error updating team members", err.Error())
		return
	}

	theme, homeUID, timezone, weekStart := teamPreferencesFromModel(plan.Preferences)
	if err := updateTeamPreferencesFromValues(client, teamID, theme, homeUID, timezone, weekStart); err != nil {
		resp.Diagnostics.AddError("Error updating team preferences", err.Error())
		return
	}

	// Team sync: diff state vs plan
	if !plan.TeamSync.IsNull() && len(plan.TeamSync.Elements()) > 0 {
		stateGroups := teamSyncGroupsFromModel(state.TeamSync)
		planGroups := teamSyncGroupsFromModel(plan.TeamSync)
		addGroups, removeGroups := sliceDiff(stateGroups, planGroups)
		if err := applyTeamExternalGroup(client, teamID, addGroups, removeGroups); err != nil {
			resp.Diagnostics.AddError("Error configuring team sync", err.Error())
			return
		}
	}

	readData, diags := r.readTeamFromID(ctx, client, teamID, &plan)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, readData)...)
}

func (r *teamResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state teamResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client, _, idFields, err := r.clientFromExistingOrgResource(resourceTeamID, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to get client", err.Error())
		return
	}
	teamIDStr := strconv.FormatInt(idFields[0].(int64), 10)

	_, err = client.Teams.DeleteTeamByID(teamIDStr)
	if err != nil && !common.IsNotFoundError(err) {
		resp.Diagnostics.AddError("Error deleting team", err.Error())
	}
}

func (r *teamResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	client, orgID, idFields, err := r.clientFromExistingOrgResource(resourceTeamID, req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Failed to get client", err.Error())
		return
	}
	teamID := idFields[0].(int64)

	team, err := getTeamByID(client, teamID)
	if err != nil {
		if common.IsNotFoundError(err) {
			resp.Diagnostics.AddError("Team not found", err.Error())
			return
		}
		resp.Diagnostics.AddError("Error reading team", err.Error())
		return
	}

	data := teamResourceModel{
		ID:                          types.StringValue(req.ID),
		OrgID:                       types.StringValue(strconv.FormatInt(orgID, 10)),
		TeamID:                      types.Int64Value(teamID),
		TeamUID:                     types.StringValue(team.UID),
		Name:                        types.StringValue(team.Name),
		Email:                       types.StringValue(team.Email),
		IgnoreExternallySyncedMembers: types.BoolValue(true),
	}
	readData, diags := r.readTeamFromID(ctx, client, teamID, &data)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, readData)...)
}

// readTeamFromID reads team from API and fills the model. If team is not found and state had no ID, returns nil, nil.
func (r *teamResource) readTeamFromID(ctx context.Context, client *goapi.GrafanaHTTPAPI, teamID int64, data *teamResourceModel) (*teamResourceModel, diag.Diagnostics) {
	team, err := getTeamByID(client, teamID)
	if err != nil {
		if common.IsNotFoundError(err) && data.ID.IsNull() {
			return nil, nil
		}
		return nil, diag.Diagnostics{diag.NewErrorDiagnostic("Error reading team", err.Error())}
	}

	orgID := team.OrgID
	data.ID = types.StringValue(MakeOrgResourceID(orgID, teamID))
	data.OrgID = types.StringValue(strconv.FormatInt(orgID, 10))
	data.TeamID = types.Int64Value(teamID)
	data.TeamUID = types.StringValue(team.UID)
	data.Name = types.StringValue(team.Name)
	data.Email = types.StringValue(team.Email)

	// Preferences
	prefsResp, err := client.Teams.GetTeamPreferences(strconv.FormatInt(teamID, 10))
	if err != nil {
		return nil, diag.Diagnostics{diag.NewErrorDiagnostic("Error reading team preferences", err.Error())}
	}
	prefs := prefsResp.GetPayload()
	if prefs.Theme+prefs.Timezone+prefs.HomeDashboardUID+prefs.WeekStart != "" {
		prefsList, d := teamPreferencesToModel(ctx, prefs.Theme, prefs.HomeDashboardUID, prefs.Timezone, prefs.WeekStart)
		if d.HasError() {
			return nil, d
		}
		data.Preferences = prefsList
	}

	// Team sync (only when present in state/config so we don't fetch unnecessarily)
	if !data.TeamSync.IsNull() {
		syncResp, err := client.SyncTeamGroups.GetTeamGroupsAPI(teamID)
		if err != nil {
			return nil, diag.Diagnostics{diag.NewErrorDiagnostic("Error reading team sync groups", err.Error())}
		}
		teamGroups := syncResp.GetPayload()
		groupStrs := make([]string, 0, len(teamGroups))
		for _, g := range teamGroups {
			groupStrs = append(groupStrs, g.GroupID)
		}
		syncList, d := teamSyncGroupsToModel(ctx, groupStrs)
		if d.HasError() {
			return nil, d
		}
		data.TeamSync = syncList
	}

	// Members
	ignoreSync := true
	if !data.IgnoreExternallySyncedMembers.IsNull() {
		ignoreSync = data.IgnoreExternallySyncedMembers.ValueBool()
	}
	members, err := readTeamMembersSlice(client, teamID, ignoreSync)
	if err != nil {
		return nil, diag.Diagnostics{diag.NewErrorDiagnostic("Error reading team members", err.Error())}
	}
	memberSet, d := types.SetValueFrom(ctx, types.StringType, members)
	if d.HasError() {
		return nil, d
	}
	data.Members = memberSet

	return data, nil
}

// readTeamFromID is used by the legacy grafana_team data source (SDK). It reads the team from the API and sets the ResourceData.
func readTeamFromID(client *goapi.GrafanaHTTPAPI, teamID int64, d *sdkschema.ResourceData, readTeamSync bool) sdkdiag.Diagnostics {
	team, err := getTeamByID(client, teamID)
	if err != nil {
		errReturn, shouldReturn := common.CheckReadError("team", d, err)
		if shouldReturn {
			return errReturn
		}
		return nil
	}

	d.SetId(MakeOrgResourceID(team.OrgID, teamID))
	d.Set("team_id", teamID)
	d.Set("team_uid", team.UID)
	d.Set("name", team.Name)
	d.Set("org_id", strconv.FormatInt(team.OrgID, 10))
	if team.Email != "" {
		d.Set("email", team.Email)
	}

	prefsResp, err := client.Teams.GetTeamPreferences(strconv.FormatInt(teamID, 10))
	if err != nil {
		return sdkdiag.FromErr(err)
	}
	prefs := prefsResp.GetPayload()
	if prefs.Theme+prefs.Timezone+prefs.HomeDashboardUID+prefs.WeekStart != "" {
		d.Set("preferences", []map[string]any{
			{
				"theme":              prefs.Theme,
				"home_dashboard_uid": prefs.HomeDashboardUID,
				"timezone":           prefs.Timezone,
				"week_start":         prefs.WeekStart,
			},
		})
	}

	if readTeamSync {
		syncResp, err := client.SyncTeamGroups.GetTeamGroupsAPI(teamID)
		if err != nil {
			return sdkdiag.FromErr(err)
		}
		teamGroups := syncResp.GetPayload()
		groupIDs := make([]string, 0, len(teamGroups))
		for _, g := range teamGroups {
			groupIDs = append(groupIDs, g.GroupID)
		}
		d.Set("team_sync", []map[string]any{{"groups": groupIDs}})
	}

	ignoreSync := true
	if v, ok := d.GetOk("ignore_externally_synced_members"); ok {
		ignoreSync = v.(bool)
	}
	members, err := readTeamMembersSlice(client, teamID, ignoreSync)
	if err != nil {
		return sdkdiag.FromErr(err)
	}
	return sdkdiag.FromErr(d.Set("members", members))
}

// Helpers that work with values (used by framework and legacy team_sync)

func setToStringSlice(set types.Set) []string {
	if set.IsNull() || set.IsUnknown() {
		return nil
	}
	var out []string
	for _, v := range set.Elements() {
		out = append(out, v.(types.String).ValueString())
	}
	return out
}

func sliceDiff(oldSlice, newSlice []string) (add, remove []string) {
	for _, s := range newSlice {
		if !slices.Contains(oldSlice, s) {
			add = append(add, s)
		}
	}
	for _, s := range oldSlice {
		if !slices.Contains(newSlice, s) {
			remove = append(remove, s)
		}
	}
	return add, remove
}

func teamPreferencesFromModel(list types.List) (theme, homeDashboardUID, timezone, weekStart string) {
	if list.IsNull() || len(list.Elements()) == 0 {
		return "", "", "", ""
	}
	elem := list.Elements()[0]
	obj, ok := elem.(types.Object)
	if !ok {
		return "", "", "", ""
	}
	attrs := obj.Attributes()
	if v, ok := attrs["theme"].(types.String); ok {
		theme = v.ValueString()
	}
	if v, ok := attrs["home_dashboard_uid"].(types.String); ok {
		homeDashboardUID = v.ValueString()
	}
	if v, ok := attrs["timezone"].(types.String); ok {
		timezone = v.ValueString()
	}
	if v, ok := attrs["week_start"].(types.String); ok {
		weekStart = v.ValueString()
	}
	return theme, homeDashboardUID, timezone, weekStart
}

var teamPreferencesObjectType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"theme":               types.StringType,
		"home_dashboard_uid": types.StringType,
		"timezone":            types.StringType,
		"week_start":          types.StringType,
	},
}

func teamPreferencesToModel(ctx context.Context, theme, homeDashboardUID, timezone, weekStart string) (types.List, diag.Diagnostics) {
	attrs := map[string]attr.Value{
		"theme":               types.StringValue(theme),
		"home_dashboard_uid": types.StringValue(homeDashboardUID),
		"timezone":            types.StringValue(timezone),
		"week_start":          types.StringValue(weekStart),
	}
	obj, d := types.ObjectValue(teamPreferencesObjectType.AttrTypes, attrs)
	if d.HasError() {
		return types.ListNull(teamPreferencesObjectType), d
	}
	return types.ListValueFrom(ctx, teamPreferencesObjectType, []attr.Value{obj})
}

func teamSyncGroupsFromModel(list types.List) []string {
	if list.IsNull() || len(list.Elements()) == 0 {
		return nil
	}
	elem := list.Elements()[0]
	obj, ok := elem.(types.Object)
	if !ok {
		return nil
	}
	groupsAttr := obj.Attributes()["groups"]
	if groupsAttr == nil {
		return nil
	}
	if set, ok := groupsAttr.(types.Set); ok {
		return setToStringSlice(set)
	}
	return nil
}

var teamSyncObjectType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"groups": types.SetType{ElemType: types.StringType},
	},
}

func teamSyncGroupsToModel(ctx context.Context, groups []string) (types.List, diag.Diagnostics) {
	groupSet, d := types.SetValueFrom(ctx, types.StringType, groups)
	if d.HasError() {
		return types.ListNull(teamSyncObjectType), d
	}
	obj, d := types.ObjectValue(teamSyncObjectType.AttrTypes, map[string]attr.Value{"groups": groupSet})
	if d.HasError() {
		return types.ListNull(teamSyncObjectType), d
	}
	return types.ListValueFrom(ctx, teamSyncObjectType, []attr.Value{obj})
}

// readTeamMembersSlice returns member emails for the team (excluding admin@localhost and optionally externally synced).
func readTeamMembersSlice(client *goapi.GrafanaHTTPAPI, teamID int64, ignoreExternallySynced bool) ([]string, error) {
	resp, err := client.Teams.GetTeamMembers(strconv.FormatInt(teamID, 10))
	if err != nil {
		return nil, err
	}
	var out []string
	for _, m := range resp.GetPayload() {
		if m.Email == "admin@localhost" {
			continue
		}
		if ignoreExternallySynced && len(m.Labels) > 0 {
			continue
		}
		out = append(out, m.Email)
	}
	return out, nil
}

func updateTeamMembersFromSlices(client *goapi.GrafanaHTTPAPI, teamID int64, stateMembers, configMembers []string) error {
	stateMap := make(map[string]TeamMember)
	configMap := make(map[string]TeamMember)
	for _, email := range stateMembers {
		if _, ok := stateMap[email]; ok {
			return fmt.Errorf("error: Team Member '%s' cannot be specified multiple times", email)
		}
		stateMap[email] = TeamMember{0, email}
	}
	for _, email := range configMembers {
		if _, ok := configMap[email]; ok {
			return fmt.Errorf("error: Team Member '%s' cannot be specified multiple times", email)
		}
		configMap[email] = TeamMember{0, email}
	}
	changes := memberChanges(stateMap, configMap)
	changes, err := addMemberIdsToChanges(client, changes)
	if err != nil {
		return err
	}
	return applyMemberChanges(client, teamID, changes)
}

func updateTeamPreferencesFromValues(client *goapi.GrafanaHTTPAPI, teamID int64, theme, homeDashboardUID, timezone, weekStart string) error {
	body := models.UpdatePrefsCmd{
		Theme:            theme,
		HomeDashboardUID: homeDashboardUID,
		Timezone:         timezone,
		WeekStart:        weekStart,
	}
	_, err := client.Teams.UpdateTeamPreferences(strconv.FormatInt(teamID, 10), &body)
	return err
}

// Shared types and helpers (used by both framework team and legacy team_external_group)

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
	gUserMap := make(map[string]int64)
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
