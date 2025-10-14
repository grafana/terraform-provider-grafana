package oncall

import (
	"context"
	"fmt"

	onCallAPI "github.com/grafana/amixr-api-go-client"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
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
		MarkdownDescription: "* [HTTP API](https://grafana.com/docs/oncall/latest/oncall-api-reference/users/)",
		Attributes: map[string]schema.Attribute{
			"username": schema.StringAttribute{
				Required:    true,
				Description: "The username of the user.",
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
		},
	}
}

func (r *userDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	// Read Terraform state data into the model
	var data userDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

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

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}
