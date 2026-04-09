package grafana

import (
	"context"
	"strconv"
	"strings"

	goapi "github.com/grafana/grafana-openapi-client-go/client"
	"github.com/grafana/grafana-openapi-client-go/client/users"
	"github.com/grafana/grafana-openapi-client-go/models"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &userResource{}
	_ resource.ResourceWithConfigure   = &userResource{}
	_ resource.ResourceWithImportState = &userResource{}
	_ resource.ResourceWithModifyPlan  = &userResource{}

	resourceUserName = "grafana_user"
	resourceUserID   = common.NewResourceID(common.IntIDField("id"))
)

func resourceUser() *common.Resource {
	return common.NewResource(
		common.CategoryGrafanaOSS,
		resourceUserName,
		resourceUserID,
		&userResource{},
	).
		WithLister(listerFunction(listUsers)).
		WithPreferredResourceNameField("login")
}

type userModel struct {
	ID       types.String `tfsdk:"id"`
	UserID   types.Int64  `tfsdk:"user_id"`
	Email    types.String `tfsdk:"email"`
	Name     types.String `tfsdk:"name"`
	Login    types.String `tfsdk:"login"`
	Password types.String `tfsdk:"password"`
	IsAdmin  types.Bool   `tfsdk:"is_admin"`
}

type userResource struct {
	basePluginFrameworkResource
}

func (r *userResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = resourceUserName
}

func (r *userResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `
* [Official documentation](https://grafana.com/docs/grafana/latest/administration/user-management/server-user-management/)
* [HTTP API](https://grafana.com/docs/grafana/latest/developer-resources/api-reference/http-api/api-legacy/user/)

This resource represents an instance-scoped resource and uses Grafana's admin APIs.
It does not work with API tokens or service accounts which are org-scoped.
You must use basic auth.
This resource is also not compatible with Grafana Cloud, as it does not allow basic auth.
`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The ID of this resource.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"user_id": schema.Int64Attribute{
				Computed:    true,
				Description: "The numerical ID of the Grafana user.",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"email": schema.StringAttribute{
				Required:    true,
				Description: "The email address of the Grafana user.",
			},
			"name": schema.StringAttribute{
				Optional:    true,
				Description: "The display name for the Grafana user.",
			},
			"login": schema.StringAttribute{
				Optional:    true,
				Description: "The username for the Grafana user.",
			},
			"password": schema.StringAttribute{
				Required:    true,
				Sensitive:   true,
				Description: "The password for the Grafana user.",
			},
			"is_admin": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether to make user an admin. Defaults to `false`.",
				Default:     booldefault.StaticBool(false),
			},
		},
	}
}

func (r *userResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.State.Raw.IsNull() || req.Plan.Raw.IsNull() {
		return
	}
	var plan, state userModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	changed := false
	if plan.Email.ValueString() != state.Email.ValueString() && strings.EqualFold(plan.Email.ValueString(), state.Email.ValueString()) {
		plan.Email = state.Email
		changed = true
	}
	if plan.Login.ValueString() != state.Login.ValueString() && strings.EqualFold(plan.Login.ValueString(), state.Login.ValueString()) {
		plan.Login = state.Login
		changed = true
	}
	if changed {
		resp.Diagnostics.Append(resp.Plan.Set(ctx, plan)...)
	}
}

func (r *userResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	readData, diags := r.read(ctx, req.ID, types.StringNull())
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if readData == nil {
		resp.Diagnostics.AddError("Resource not found", "User not found")
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, readData)...)
}

func (r *userResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan userModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client, err := r.globalClient()
	if err != nil {
		resp.Diagnostics.AddError(err.Error(), "")
		return
	}

	user := models.AdminCreateUserForm{
		Email:    plan.Email.ValueString(),
		Name:     plan.Name.ValueString(),
		Login:    plan.Login.ValueString(),
		Password: models.Password(plan.Password.ValueString()),
	}
	createResp, err := client.AdminUsers.AdminCreateUser(&user)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create user", err.Error())
		return
	}
	id := createResp.Payload.ID

	if plan.IsAdmin.ValueBool() {
		perm := models.AdminUpdateUserPermissionsForm{IsGrafanaAdmin: true}
		if _, err = client.AdminUsers.AdminUpdateUserPermissions(id, &perm); err != nil {
			resp.Diagnostics.AddError("Failed to set admin permission", err.Error())
			return
		}
	}

	idStr := strconv.FormatInt(id, 10)
	readData, diags := r.read(ctx, idStr, plan.Password)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, readData)...)
}

func (r *userResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state userModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	readData, diags := r.read(ctx, state.ID.ValueString(), state.Password)
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

func (r *userResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state userModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client, err := r.globalClient()
	if err != nil {
		resp.Diagnostics.AddError(err.Error(), "")
		return
	}

	id, err := strconv.ParseInt(plan.ID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid user ID", err.Error())
		return
	}

	u := models.UpdateUserCommand{
		Email: plan.Email.ValueString(),
		Name:  plan.Name.ValueString(),
		Login: plan.Login.ValueString(),
	}
	if _, err = client.Users.UpdateUser(id, &u); err != nil {
		resp.Diagnostics.AddError("Failed to update user", err.Error())
		return
	}

	if !plan.Password.IsNull() && !plan.Password.IsUnknown() && plan.Password.ValueString() != "" {
		f := models.AdminUpdateUserPasswordForm{Password: models.Password(plan.Password.ValueString())}
		if _, err = client.AdminUsers.AdminUpdateUserPassword(id, &f); err != nil {
			resp.Diagnostics.AddError("Failed to update password", err.Error())
			return
		}
	}

	if !plan.IsAdmin.Equal(state.IsAdmin) && !plan.IsAdmin.IsNull() && !plan.IsAdmin.IsUnknown() {
		perm := models.AdminUpdateUserPermissionsForm{IsGrafanaAdmin: plan.IsAdmin.ValueBool()}
		if _, err = client.AdminUsers.AdminUpdateUserPermissions(id, &perm); err != nil {
			resp.Diagnostics.AddError("Failed to update admin permission", err.Error())
			return
		}
	}

	readData, diags := r.read(ctx, plan.ID.ValueString(), plan.Password)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, readData)...)
}

func (r *userResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state userModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client, err := r.globalClient()
	if err != nil {
		resp.Diagnostics.AddError(err.Error(), "")
		return
	}

	id, err := strconv.ParseInt(state.ID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid user ID", err.Error())
		return
	}

	_, err = client.AdminUsers.AdminDeleteUser(id)
	if err != nil && !common.IsNotFoundError(err) {
		resp.Diagnostics.AddError("Failed to delete user", err.Error())
	}
}

func (r *userResource) read(ctx context.Context, idStr string, passwordState types.String) (*userModel, diag.Diagnostics) {
	var diags diag.Diagnostics
	client, err := r.globalClient()
	if err != nil {
		diags.AddError(err.Error(), "")
		return nil, diags
	}

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		diags.AddError("Invalid user ID", err.Error())
		return nil, diags
	}

	resp, err := client.Users.GetUserByID(id)
	if err != nil {
		if common.IsNotFoundError(err) {
			return nil, diags
		}
		diags.AddError("Failed to read user", err.Error())
		return nil, diags
	}
	user := resp.Payload

	nameVal := types.StringNull()
	if user.Name != "" {
		nameVal = types.StringValue(user.Name)
	}
	loginVal := types.StringNull()
	if user.Login != "" {
		loginVal = types.StringValue(user.Login)
	}

	return &userModel{
		ID:       types.StringValue(strconv.FormatInt(user.ID, 10)),
		UserID:   types.Int64Value(user.ID),
		Email:    types.StringValue(user.Email),
		Name:     nameVal,
		Login:    loginVal,
		Password: passwordState,
		IsAdmin:  types.BoolValue(user.IsGrafanaAdmin),
	}, diags
}

func listUsers(ctx context.Context, client *goapi.GrafanaHTTPAPI, data *ListerData) ([]string, error) {
	api := client.Clone().WithOrgID(0)
	var ids []string
	var page int64 = 1
	for {
		params := users.NewSearchUsersParams().WithPage(&page)
		resp, err := api.Users.SearchUsers(params)
		if err != nil {
			return nil, err
		}
		for _, user := range resp.Payload {
			ids = append(ids, strconv.FormatInt(user.ID, 10))
		}
		if len(resp.Payload) == 0 {
			break
		}
		page++
	}
	return ids, nil
}
