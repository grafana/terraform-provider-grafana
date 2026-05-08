package grafana

import (
	"context"
	"strconv"
	"strings"

	"github.com/grafana/grafana-openapi-client-go/client/teams"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_                        datasource.DataSourceWithConfigure = (*teamsDataSource)(nil)
	dataSourceTeamsTypeName                                     = "grafana_teams"
)

func datasourceTeams() *common.DataSource {
	return common.NewDataSource(
		common.CategoryGrafanaOSS,
		dataSourceTeamsTypeName,
		&teamsDataSource{},
	)
}

type teamsDataSourceTeamModel struct {
	ID          types.Int64  `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	UID         types.String `tfsdk:"uid"`
	Email       types.String `tfsdk:"email"`
	MemberCount types.Int64  `tfsdk:"member_count"`
	OrgID       types.Int64  `tfsdk:"org_id"`
}

type teamsDataSourceModel struct {
	ID    types.String `tfsdk:"id"`
	OrgID types.String `tfsdk:"org_id"`
	Query types.String `tfsdk:"query"`
	Teams []teamsDataSourceTeamModel `tfsdk:"teams"`
}

type teamsDataSource struct {
	basePluginFrameworkDataSource
}

func (d *teamsDataSource) Metadata(_ context.Context, _ datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = dataSourceTeamsTypeName
}

func (d *teamsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `
Fetches a list of teams from Grafana, optionally filtered by a search keyword.

* [Official documentation](https://grafana.com/docs/grafana/latest/administration/team-management/)
* [HTTP API](https://grafana.com/docs/grafana/latest/developer-resources/api-reference/http-api/api-legacy/team/)
`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
			},
			"org_id": pluginFrameworkOrgIDAttribute(),
			"query": schema.StringAttribute{
				Optional:    true,
				Description: "A keyword to filter teams by name (substring match). If omitted, all teams are returned.",
			},
			"teams": schema.ListAttribute{
				Computed:    true,
				Description: "The list of matching Grafana teams.",
				ElementType: types.ObjectType{
					AttrTypes: map[string]attr.Type{
						"id":           types.Int64Type,
						"name":         types.StringType,
						"uid":          types.StringType,
						"email":        types.StringType,
						"member_count": types.Int64Type,
						"org_id":       types.Int64Type,
					},
				},
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

	var page int64 = 1
	data.Teams = make([]teamsDataSourceTeamModel, 0)

	for {
		params := teams.NewSearchTeamsParams().WithPage(&page)
		if !data.Query.IsNull() && data.Query.ValueString() != "" {
			query := data.Query.ValueString()
			params = params.WithQuery(&query)
		}

		searchResp, err := client.Teams.SearchTeams(params)
		if err != nil {
			resp.Diagnostics.AddError("Failed to search teams", err.Error())
			return
		}

		payload := searchResp.GetPayload()
		for _, t := range payload.Teams {
			data.Teams = append(data.Teams, teamsDataSourceTeamModel{
				ID:          types.Int64Value(*t.ID),
				Name:        types.StringValue(*t.Name),
				UID:         types.StringValue(*t.UID),
				Email:       types.StringValue(t.Email),
				MemberCount: types.Int64Value(*t.MemberCount),
				OrgID:       types.Int64Value(*t.OrgID),
			})
		}

		if int64(len(data.Teams)) >= payload.TotalCount {
			break
		}
		page++
	}

	// Build a deterministic ID from the query parameters.
	idParts := []string{strconv.FormatInt(orgID, 10)}
	if !data.Query.IsNull() && data.Query.ValueString() != "" {
		idParts = append(idParts, data.Query.ValueString())
	}
	data.ID = types.StringValue(strings.Join(idParts, ":"))

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}
