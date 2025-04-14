package oncall

import (
	"context"

	onCallAPI "github.com/grafana/amixr-api-go-client"
	"github.com/grafana/terraform-provider-grafana/v3/internal/common"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
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
		MarkdownDescription: "* [HTTP API](https://grafana.com/docs/oncall/latest/oncall-api-reference/users/)",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
			},
			"users": schema.ListAttribute{
				ElementType: types.ObjectType{
					AttrTypes: map[string]attr.Type{
						"id":       types.StringType,
						"username": types.StringType,
						"email":    types.StringType,
						"role":     types.StringType,
					},
				},
				Computed: true,
			},
		},
	}
}

type userDataSourceModel struct {
	ID       basetypes.StringValue `tfsdk:"id"`
	Username basetypes.StringValue `tfsdk:"username"`
	Email    basetypes.StringValue `tfsdk:"email"`
	Role     basetypes.StringValue `tfsdk:"role"`
}

type usersDataSourceModel struct {
	ID    basetypes.StringValue `tfsdk:"id"`
	Users []userDataSourceModel `tfsdk:"users"`
}

func (r *usersDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	// Read Terraform state data into the model
	var data usersDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	allUsers := []userDataSourceModel{}
	page := 1
	for {
		options := &onCallAPI.ListUserOptions{
			ListOptions: onCallAPI.ListOptions{
				Page: page,
			},
		}
		usersResponse, _, err := r.client.Users.ListUsers(options)
		if err != nil {
			resp.Diagnostics.AddError("Failed to list users", err.Error())
			return
		}

		for _, user := range usersResponse.Users {
			allUsers = append(allUsers, userDataSourceModel{
				ID:       basetypes.NewStringValue(user.ID),
				Username: basetypes.NewStringValue(user.Username),
				Email:    basetypes.NewStringValue(user.Email),
				Role:     basetypes.NewStringValue(user.Role),
			})
		}

		if usersResponse.PaginatedResponse.Next == nil {
			break
		}
		page++
	}

	data.ID = basetypes.NewStringValue("oncall_users") // singleton
	data.Users = allUsers

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}
