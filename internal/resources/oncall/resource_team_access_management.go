package oncall

import (
	"context"
	"net/http"

	onCallAPI "github.com/grafana/amixr-api-go-client"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &teamAccessManagementResource{}
	_ resource.ResourceWithConfigure   = &teamAccessManagementResource{}
	_ resource.ResourceWithImportState = &teamAccessManagementResource{}
)

func resourceTeamAccessManagement() *common.Resource {
	return common.NewResource(
		common.CategoryOnCall,
		"grafana_oncall_team_access_management",
		resourceID,
		&teamAccessManagementResource{},
	)
}

type teamAccessManagementResourceModel struct {
	ID                      types.String `tfsdk:"id"`
	TeamID                  types.String `tfsdk:"team_id"`
	IsSharingResourcesToAll types.Bool   `tfsdk:"is_sharing_resources_to_all"`
}

type teamAccessManagementResource struct {
	basePluginFrameworkResource
}

func (r *teamAccessManagementResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "grafana_oncall_team_access_management"
}

func (r *teamAccessManagementResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `Manages the Team Access Management setting of a Grafana OnCall team.

Destroying this resource only removes it from Terraform state; the setting on the team is left unchanged.

* [Official documentation](https://grafana.com/docs/oncall/latest/set-up/manage-access/teams/)
* [HTTP API](https://grafana.com/docs/oncall/latest/oncall-api-reference/teams/)`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"team_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The ID of the OnCall team, for example from the `grafana_oncall_team` data source.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"is_sharing_resources_to_all": schema.BoolAttribute{
				Required:            true,
				MarkdownDescription: "Whether the team's resources are visible to all users (`true`), or only to team members and admins (`false`).",
			},
		},
	}
}

func (r *teamAccessManagementResource) update(plan *teamAccessManagementResourceModel) error {
	team, _, err := r.client.Teams.UpdateTeam(plan.TeamID.ValueString(), &onCallAPI.UpdateTeamOptions{
		IsSharingResourcesToAll: plan.IsSharingResourcesToAll.ValueBool(),
	})
	if err != nil {
		return err
	}
	plan.ID = types.StringValue(team.ID)
	plan.IsSharingResourcesToAll = types.BoolValue(team.IsSharingResourcesToAll)
	return nil
}

func (r *teamAccessManagementResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan teamAccessManagementResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.update(&plan); err != nil {
		resp.Diagnostics.AddError("Failed to update team access management", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *teamAccessManagementResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state teamAccessManagementResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	team, httpResp, err := r.client.Teams.GetTeam(state.TeamID.ValueString(), &onCallAPI.GetTeamOptions{})
	if err != nil {
		if httpResp != nil && httpResp.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read team", err.Error())
		return
	}

	state.ID = types.StringValue(team.ID)
	state.IsSharingResourcesToAll = types.BoolValue(team.IsSharingResourcesToAll)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *teamAccessManagementResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan teamAccessManagementResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.update(&plan); err != nil {
		resp.Diagnostics.AddError("Failed to update team access management", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *teamAccessManagementResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// The setting cannot be removed from a team, so deleting only drops the resource from state.
}

func (r *teamAccessManagementResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("team_id"), req.ID)...)
}
