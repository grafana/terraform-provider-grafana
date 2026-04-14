package grafana

import (
	"context"
	"strconv"

	"github.com/grafana/grafana-openapi-client-go/client/teams"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSourceWithConfigure = (*teamsDataSource)(nil)

func datasourceTeams() *common.DataSource {
	return common.NewDataSource(
		common.CategoryGrafanaOSS,
		"grafana_teams",
		&teamsDataSource{},
	)
}

type teamsDataSourceModel struct {
	ID    types.String     `tfsdk:"id"`
	OrgID types.String     `tfsdk:"org_id"`
	Teams []teamsTeamModel `tfsdk:"teams"`
}

type teamsTeamModel struct {
	TeamID  types.Int64  `tfsdk:"team_id"`
	TeamUID types.String `tfsdk:"team_uid"`
	Name    types.String `tfsdk:"name"`
	Email   types.String `tfsdk:"email"`
}

type teamsDataSource struct {
	basePluginFrameworkDataSource
}

func (d *teamsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = "grafana_teams"
}

func (d *teamsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `
* [Official documentation](https://grafana.com/docs/grafana/latest/administration/team-management/)
* [HTTP API](https://grafana.com/docs/grafana/latest/developers/http_api/team/)
`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
			},
			"org_id": pluginFrameworkOrgIDAttribute(),
			"teams": schema.ListAttribute{
				Computed: true,
				ElementType: types.ObjectType{
					AttrTypes: map[string]attr.Type{
						"team_id":  types.Int64Type,
						"team_uid": types.StringType,
						"name":     types.StringType,
						"email":    types.StringType,
					},
				},
				Description: "A list of Grafana teams.",
			},
		},
	}
}

func (d *teamsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data teamsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client, orgID, err := d.clientFromNewOrgResource(data.OrgID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to get client", err.Error())
		return
	}

	var allTeams []teamsTeamModel
	var page int64 = 1

	for {
		params := teams.NewSearchTeamsParams().WithPage(&page)
		searchResp, err := client.Teams.SearchTeams(params)
		if err != nil {
			resp.Diagnostics.AddError("Failed to search teams", err.Error())
			return
		}

		for _, t := range searchResp.GetPayload().Teams {
			allTeams = append(allTeams, teamsTeamModel{
				TeamID:  types.Int64Value(t.ID),
				TeamUID: types.StringValue(t.UID),
				Name:    types.StringValue(t.Name),
				Email:   types.StringValue(t.Email),
			})
		}

		if searchResp.GetPayload().TotalCount <= int64(len(allTeams)) {
			break
		}

		page++
	}

	data.ID = types.StringValue(strconv.FormatInt(orgID, 10))
	data.OrgID = types.StringValue(strconv.FormatInt(orgID, 10))
	data.Teams = allTeams

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}
