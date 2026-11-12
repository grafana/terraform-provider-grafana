package oncall

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	onCallAPI "github.com/grafana/amixr-api-go-client"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

var dataSourceUsersName = "grafana_oncall_users"

func dataSourceUsers() *common.DataSource {
	return common.NewDataSource(common.CategoryOnCall, dataSourceUsersName, &usersDataSource{})
}

type usersDataSource struct {
	basePluginFrameworkDataSource
}

func (r *usersDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = dataSourceUsersName
}

func (r *usersDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists Grafana OnCall users. Without `team_id`, every plan reads the full user list from the API and stores it in state, so in organizations with many users every plan makes more API requests. To look up a single user, use `grafana_oncall_user`.\n\n" +
			"Each user's `username` is their Grafana login, which can differ from their email address. To refer to users by email, build a map from `email` to `id`, as shown in the example.\n\n" +
			"Team membership is synced from Grafana and the sync is not immediate, so a user added to a team moments earlier can be missing from the first plan.\n\n" +
			"Users are returned in OnCall's own order, not by name. A rotation built from team membership therefore re-orders when the team changes: keep the order in the configuration and use membership to add and remove people, as shown in the example.\n\n" +
			"* [HTTP API](https://grafana.com/docs/oncall/latest/oncall-api-reference/users/)",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
			},
			"team_id": schema.StringAttribute{
				Optional:    true,
				Description: "Return only the members of this team, using the ID from the `grafana_oncall_team` data source. A team ID that does not exist returns no users instead of an error, so reference the data source rather than writing an ID by hand. Unlike `grafana_oncall_integration`, this attribute does not accept `\"null\"`: there is no filter for users who belong to no team.",
				Validators: []validator.String{
					// A blank filter would be dropped from the request and read every
					// user, which looks just like a team with no members.
					stringvalidator.RegexMatches(regexp.MustCompile(`\S`),
						"must be a team ID, such as the id of the `grafana_oncall_team` data source, or omitted entirely to read every user"),
				},
			},
			"users": schema.ListAttribute{
				Description: "Every matching OnCall user, with their ID, username (the Grafana login), email, and role. `teams` holds the IDs of the teams the user belongs to, which are the IDs the `grafana_oncall_team` data source returns.",
				ElementType: types.ObjectType{
					AttrTypes: map[string]attr.Type{
						"id":       types.StringType,
						"username": types.StringType,
						"email":    types.StringType,
						"role":     types.StringType,
						"teams":    types.ListType{ElemType: types.StringType},
					},
				},
				Computed: true,
			},
		},
	}
}

// Each element of users is a userDataSourceModel, declared in data_source_user.go
// and shared with the singular data source so the two cannot describe a user
// differently. Both this element type and that schema must keep exactly the
// attributes that model declares; TestUnitUserModelMatchesSchemas enforces it.
type usersDataSourceModel struct {
	ID     basetypes.StringValue `tfsdk:"id"`
	TeamID basetypes.StringValue `tfsdk:"team_id"`
	Users  []userDataSourceModel `tfsdk:"users"`
}

func (r *usersDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	// Read Terraform state data into the model
	var data usersDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// A filter that only resolves at apply time escapes the validator, which skips
	// values that are unknown at plan. Left unchecked, a blank one is dropped from
	// the request and silently returns every user.
	if !data.TeamID.IsNull() {
		switch {
		case strings.TrimSpace(data.TeamID.ValueString()) == "":
			resp.Diagnostics.AddAttributeError(path.Root("team_id"), "Blank team ID",
				"team_id resolved to a blank string. Remove the attribute to read every user, or set it to the ID of a team.")
			return
		case data.TeamID.ValueString() == onCallAPI.NoTeamID:
			resp.Diagnostics.AddAttributeError(path.Root("team_id"), "Unsupported team ID",
				`team_id does not accept "null". Unlike grafana_oncall_integration, the users API has no filter for users who belong to no team, and "null" matches no users at all.`)
			return
		}
	}

	allUsers := []userDataSourceModel{}
	page := 1
	for {
		options := &onCallAPI.ListUserOptions{
			ListOptions: onCallAPI.ListOptions{
				Page: page,
			},
			// Unset when team_id is not configured, which lists every user.
			TeamID: data.TeamID.ValueString(),
		}
		usersResponse, _, err := r.client.Users.ListUsers(options)
		if err != nil {
			resp.Diagnostics.AddError("Failed to list users", err.Error())
			return
		}

		for _, user := range usersResponse.Users {
			teams, diags := userTeamsValue(ctx, user)
			resp.Diagnostics.Append(diags...)
			if resp.Diagnostics.HasError() {
				return
			}

			allUsers = append(allUsers, userDataSourceModel{
				ID:       basetypes.NewStringValue(user.ID),
				Username: basetypes.NewStringValue(user.Username),
				Email:    basetypes.NewStringValue(user.Email),
				Role:     basetypes.NewStringValue(user.Role),
				Teams:    teams,
			})
		}

		if usersResponse.PaginatedResponse.Next == nil {
			break
		}
		page++
	}

	// One instance per filter: unfiltered reads keep the original singleton ID.
	dataSourceID := "oncall_users"
	if teamID := data.TeamID.ValueString(); teamID != "" {
		dataSourceID = "oncall_users:" + teamID

		// A rotation built from no users pages nobody, and nothing else reports it:
		// an ID that matches no team returns an empty list, the same as a real team
		// with no members.
		if len(allUsers) == 0 {
			resp.Diagnostics.AddAttributeWarning(path.Root("team_id"), "No users returned",
				fmt.Sprintf("No users were returned for team ID %s, so anything built from this data source, such as a rotation, will be empty. "+
					"Either no team has this ID, or the team has no members. "+
					"Membership is synced from Grafana and the sync is not immediate, so members added in the last minute may not appear yet.", teamID))
		}
	}
	data.ID = basetypes.NewStringValue(dataSourceID)
	data.Users = allUsers

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}
