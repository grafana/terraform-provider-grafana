package grafana

import (
	"context"
	"fmt"
	"strconv"

	goapi "github.com/grafana/grafana-openapi-client-go/client"
	"github.com/grafana/grafana-openapi-client-go/client/teams"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSourceWithConfigure = (*teamDataSource)(nil)

func datasourceTeam() *common.DataSource {
	return common.NewDataSource(
		common.CategoryGrafanaOSS,
		"grafana_team",
		&teamDataSource{},
	)
}

type teamDataSourceModel struct {
	ID           types.String       `tfsdk:"id"`
	OrgID        types.String       `tfsdk:"org_id"`
	Name         types.String       `tfsdk:"name"`
	TeamID       types.Int64        `tfsdk:"team_id"`
	TeamUID      types.String       `tfsdk:"team_uid"`
	Email        types.String       `tfsdk:"email"`
	Members      types.Set          `tfsdk:"members"`
	ReadTeamSync types.Bool         `tfsdk:"read_team_sync"`
	Preferences  []dsTeamPrefsBlock `tfsdk:"preferences"`
	TeamSync     []dsTeamSyncBlock  `tfsdk:"team_sync"`
}

type dsTeamPrefsBlock struct {
	Theme            types.String `tfsdk:"theme"`
	HomeDashboardUID types.String `tfsdk:"home_dashboard_uid"`
	Timezone         types.String `tfsdk:"timezone"`
	WeekStart        types.String `tfsdk:"week_start"`
}

type dsTeamSyncBlock struct {
	Groups types.Set `tfsdk:"groups"`
}

type teamDataSource struct {
	basePluginFrameworkDataSource
}

func (d *teamDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = "grafana_team"
}

func (d *teamDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `
* [Official documentation](https://grafana.com/docs/grafana/latest/administration/team-management/)
* [HTTP API](https://grafana.com/docs/grafana/latest/developer-resources/api-reference/http-api/api-legacy/team/)
`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
			},
			"org_id": pluginFrameworkOrgIDAttribute(),
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the Grafana team.",
			},
			"team_id": schema.Int64Attribute{
				Computed:    true,
				Description: "The team id assigned to this team by Grafana.",
			},
			"team_uid": schema.StringAttribute{
				Computed:    true,
				Description: "The team uid assigned to this team by Grafana.",
			},
			"email": schema.StringAttribute{
				Computed:    true,
				Description: "An email address for the team.",
			},
			"members": schema.SetAttribute{
				ElementType: types.StringType,
				Computed:    true,
				Description: "A set of email addresses corresponding to users who are members of the team.",
			},
			"read_team_sync": schema.BoolAttribute{
				Optional:    true,
				Description: "Whether to read the team sync settings. This is only available in Grafana Enterprise.",
			},
		},
		Blocks: map[string]schema.Block{
			"preferences": schema.ListNestedBlock{
				Description: "Team preferences.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"theme": schema.StringAttribute{
							Computed:    true,
							Description: "The default theme for this team.",
						},
						"home_dashboard_uid": schema.StringAttribute{
							Computed:    true,
							Description: "The UID of the dashboard to display when a team member logs in.",
						},
						"timezone": schema.StringAttribute{
							Computed:    true,
							Description: "The default timezone for this team.",
						},
						"week_start": schema.StringAttribute{
							Computed:    true,
							Description: "The default week start day for this team.",
						},
					},
				},
			},
			"team_sync": schema.ListNestedBlock{
				Description: "Sync external auth provider groups with this Grafana team. Only available in Grafana Enterprise.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"groups": schema.SetAttribute{
							ElementType: types.StringType,
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (d *teamDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data teamDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client, orgID, err := d.clientFromNewOrgResource(data.OrgID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to get client", err.Error())
		return
	}

	name := data.Name.ValueString()
	params := teams.NewSearchTeamsParams().WithName(&name)
	searchResp, err := client.Teams.SearchTeams(params)
	if err != nil {
		resp.Diagnostics.AddError("Failed to search teams", err.Error())
		return
	}

	var teamID int64
	found := false
	for _, t := range searchResp.GetPayload().Teams {
		if t.Name != nil && *t.Name == name {
			teamID = *t.ID
			found = true
			break
		}
	}
	if !found {
		resp.Diagnostics.AddError("Team not found", fmt.Sprintf("no team with name %q", name))
		return
	}

	team, err := getTeamByID(client, teamID)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read team", err.Error())
		return
	}

	data.ID = types.StringValue(MakeOrgResourceID(orgID, strconv.FormatInt(teamID, 10)))
	data.OrgID = types.StringValue(strconv.FormatInt(orgID, 10))
	data.TeamID = types.Int64Value(teamID)
	data.TeamUID = types.StringValue(*team.UID)
	data.Name = types.StringValue(*team.Name)
	data.Email = types.StringValue(team.Email)

	// Preferences
	prefsResp, err := client.Teams.GetTeamPreferences(strconv.FormatInt(teamID, 10))
	if err != nil {
		resp.Diagnostics.AddError("Failed to read team preferences", err.Error())
		return
	}
	prefs := prefsResp.GetPayload()
	if prefs.Theme != "" || prefs.Timezone != "" || prefs.HomeDashboardUID != "" || prefs.WeekStart != "" {
		data.Preferences = []dsTeamPrefsBlock{{
			Theme:            types.StringValue(prefs.Theme),
			HomeDashboardUID: types.StringValue(prefs.HomeDashboardUID),
			Timezone:         types.StringValue(prefs.Timezone),
			WeekStart:        types.StringValue(prefs.WeekStart),
		}}
	}

	// Team sync — only fetched when read_team_sync is explicitly true
	if !data.ReadTeamSync.IsNull() && data.ReadTeamSync.ValueBool() {
		syncResp, err := client.SyncTeamGroups.GetTeamGroupsAPI(strconv.FormatInt(teamID, 10))
		if err != nil {
			resp.Diagnostics.AddError("Failed to read team sync groups", err.Error())
			return
		}
		groupStrs := make([]string, 0, len(syncResp.GetPayload()))
		for _, g := range syncResp.GetPayload() {
			groupStrs = append(groupStrs, g.GroupID)
		}
		groupSet, diags := types.SetValueFrom(ctx, types.StringType, groupStrs)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		data.TeamSync = []dsTeamSyncBlock{{Groups: groupSet}}
	}

	// Members — data source always ignores externally synced members (matching original behavior)
	members, err := dsReadTeamMembersSlice(client, teamID)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read team members", err.Error())
		return
	}
	memberSet, diags := types.SetValueFrom(ctx, types.StringType, members)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.Members = memberSet

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

// dsReadTeamMembersSlice returns member emails, excluding admin@localhost and externally synced members.
func dsReadTeamMembersSlice(client *goapi.GrafanaHTTPAPI, teamID int64) ([]string, error) {
	resp, err := client.Teams.GetTeamMembers(strconv.FormatInt(teamID, 10))
	if err != nil {
		return nil, err
	}
	out := make([]string, 0)
	for _, m := range resp.GetPayload() {
		if m.Email == "admin@localhost" {
			continue
		}
		if len(m.Labels) > 0 {
			continue
		}
		out = append(out, m.Email)
	}
	return out, nil
}
