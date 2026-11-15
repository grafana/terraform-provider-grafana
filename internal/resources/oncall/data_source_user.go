package oncall

import (
	"context"
	"fmt"

	onCallAPI "github.com/grafana/amixr-api-go-client"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

var dataSourceUserName = "grafana_oncall_user"

func dataSourceUser() *common.DataSource {
	return common.NewDataSource(common.CategoryOnCall, dataSourceUserName, &userDataSource{})
}

type userDataSource struct {
	basePluginFrameworkDataSource
}

func (r *userDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = dataSourceUserName
}

func (r *userDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Looks up a single Grafana OnCall user by username. To look up many users at once, use `grafana_oncall_users`.\n\n" +
			"* [HTTP API](https://grafana.com/docs/oncall/latest/oncall-api-reference/users/)",
		Attributes: map[string]schema.Attribute{
			"username": schema.StringAttribute{
				Required:    true,
				Description: "The user's Grafana login, which can differ from their email address. Matching is case-insensitive.",
			},
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The ID of the user.",
			},
			"email": schema.StringAttribute{
				Computed:    true,
				Description: "The email of the user.",
			},
			"role": schema.StringAttribute{
				Computed:    true,
				Description: "The role of the user.",
			},
			"teams": schema.ListAttribute{
				Computed:    true,
				ElementType: types.StringType,
				Description: "The IDs of the teams the user belongs to, which are the IDs the `grafana_oncall_team` data source returns. Empty when the user belongs to no team.",
			},
		},
	}
}

// userDataSourceModel describes an OnCall user everywhere the provider returns
// one: it is the whole state of grafana_oncall_user, and one element of the list
// grafana_oncall_users returns. The framework requires a model and its object to
// match field for attribute, so a field added here needs the same attribute in
// this file's schema and in the users element type in data_source_users.go.
// TestUnitUserModelMatchesSchemas fails if the two ever drift apart.
type userDataSourceModel struct {
	ID       basetypes.StringValue `tfsdk:"id"`
	Username basetypes.StringValue `tfsdk:"username"`
	Email    basetypes.StringValue `tfsdk:"email"`
	Role     basetypes.StringValue `tfsdk:"role"`
	Teams    basetypes.ListValue   `tfsdk:"teams"`
}

// userTeamsValue converts a user's team IDs for Terraform state. A user on no
// team gets an empty list rather than null, so that contains() and other list
// functions work without a null check.
func userTeamsValue(ctx context.Context, user *onCallAPI.User) (basetypes.ListValue, diag.Diagnostics) {
	teams := user.Teams
	if teams == nil {
		teams = []string{}
	}
	return types.ListValueFrom(ctx, types.StringType, teams)
}

func (r *userDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	// Read Terraform state data into the model
	var data userDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		// Without this, username is null, every user is listed, and the real
		// failure is buried under "more than one user found matching: ".
		return
	}

	options := &onCallAPI.ListUserOptions{
		Username: data.Username.ValueString(),
	}
	usersResponse, _, err := r.client.Users.ListUsers(options)
	if err != nil {
		resp.Diagnostics.AddError("Failed to list users", err.Error())
		return
	}

	if len(usersResponse.Users) == 0 {
		resp.Diagnostics.AddError("user not found", fmt.Sprintf("couldn't find a user matching: %s", options.Username))
		return
	} else if len(usersResponse.Users) != 1 {
		resp.Diagnostics.AddError("more than one user found", fmt.Sprintf("more than one user found matching: %s", options.Username))
		return
	}

	user := usersResponse.Users[0]
	data.ID = basetypes.NewStringValue(user.ID)
	data.Email = basetypes.NewStringValue(user.Email)
	data.Role = basetypes.NewStringValue(user.Role)

	teams, diags := userTeamsValue(ctx, user)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.Teams = teams

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}
