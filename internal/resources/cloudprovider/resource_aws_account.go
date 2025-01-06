package cloudprovider

import (
	"context"
	"fmt"
	"strings"

	"github.com/grafana/terraform-provider-grafana/v3/internal/common"
	"github.com/grafana/terraform-provider-grafana/v3/internal/common/cloudproviderapi"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

var (
	resourceAWSAccountTerraformName = "grafana_cloud_provider_aws_account"
	resourceAWSAccountTerraformID   = common.NewResourceID(common.StringIDField("stack_id"), common.StringIDField("resource_id"))
)

type resourceAWSAccountModel struct {
	ID         types.String `tfsdk:"id"`
	StackID    types.String `tfsdk:"stack_id"`
	ResourceID types.String `tfsdk:"resource_id"`
	Name       types.String `tfsdk:"name"`
	RoleARN    types.String `tfsdk:"role_arn"`
	Regions    types.Set    `tfsdk:"regions"`
}

type resourceAWSAccount struct {
	client *cloudproviderapi.Client
}

func makeResourceAWSAccount() *common.Resource {
	return common.NewResource(
		common.CategoryCloudProvider,
		resourceAWSAccountTerraformName,
		resourceAWSAccountTerraformID,
		&resourceAWSAccount{},
	)
}

func (r *resourceAWSAccount) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Configure is called multiple times (sometimes when ProviderData is not yet available), we only want to configure once
	if req.ProviderData == nil || r.client != nil {
		return
	}

	client, err := withClientForResource(req, resp)
	if err != nil {
		return
	}

	r.client = client
}

func (r resourceAWSAccount) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = resourceAWSAccountTerraformName
}

func (r resourceAWSAccount) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The Terraform Resource ID. This has the format \"{{ stack_id }}:{{ resource_id }}\".",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					// See https://developer.hashicorp.com/terraform/plugin/framework/resources/plan-modification#usestateforunknown
					// for details on how this works.
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"stack_id": schema.StringAttribute{
				Description: "The StackID of the Grafana Cloud instance. Part of the Terraform Resource ID.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"resource_id": schema.StringAttribute{
				Description: "The ID given by the Grafana Cloud Provider API to this AWS Account resource.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					// See https://developer.hashicorp.com/terraform/plugin/framework/resources/plan-modification#usestateforunknown
					// for details on how this works.
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "An optional human-readable name for this AWS Account resource.",
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"role_arn": schema.StringAttribute{
				Description: "An IAM Role ARN string to represent with this AWS Account resource.",
				Required:    true,
			},
			"regions": schema.SetAttribute{
				Description: "A set of regions that this AWS Account resource applies to.",
				Required:    true,
				Validators: []validator.Set{
					setvalidator.SizeAtLeast(1),
				},
				ElementType: types.StringType,
			},
		},
	}
}

func (r resourceAWSAccount) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, ":", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Invalid ID: %s", req.ID))
		return
	}
	stackID := parts[0]
	resourceID := parts[1]

	account, err := r.client.GetAWSAccount(
		ctx,
		stackID,
		resourceID,
	)
	if err != nil {
		resp.Diagnostics.AddError("Failed to get AWS Account", err.Error())
		return
	}

	regions, diags := types.SetValueFrom(ctx, basetypes.StringType{}, account.Regions)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.State.Set(ctx, &resourceAWSAccountModel{
		ID:         types.StringValue(req.ID),
		StackID:    types.StringValue(stackID),
		ResourceID: types.StringValue(resourceID),
		Name:       types.StringValue(account.Name),
		RoleARN:    types.StringValue(account.RoleARN),
		Regions:    regions,
	})
}

func (r resourceAWSAccount) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data resourceAWSAccountModel
	diags := req.Plan.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	accountData := cloudproviderapi.AWSAccount{}
	accountData.RoleARN = data.RoleARN.ValueString()
	accountData.Name = data.Name.ValueString()
	diags = data.Regions.ElementsAs(ctx, &accountData.Regions, false)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	account, err := r.client.CreateAWSAccount(
		ctx,
		data.StackID.ValueString(),
		accountData,
	)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create AWS Account", err.Error())
		return
	}

	resp.State.Set(ctx, &resourceAWSAccountModel{
		ID:         types.StringValue(resourceAWSAccountTerraformID.Make(data.StackID.ValueString(), account.ID)),
		StackID:    data.StackID,
		ResourceID: types.StringValue(account.ID),
		Name:       data.Name,
		RoleARN:    data.RoleARN,
		Regions:    data.Regions,
	})
}

func (r resourceAWSAccount) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data resourceAWSAccountModel
	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	account, err := r.client.GetAWSAccount(
		ctx,
		data.StackID.ValueString(),
		data.ResourceID.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError("Failed to get AWS Account", err.Error())
		return
	}

	diags = resp.State.SetAttribute(ctx, path.Root("role_arn"), account.RoleARN)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	diags = resp.State.SetAttribute(ctx, path.Root("name"), account.Name)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	diags = resp.State.SetAttribute(ctx, path.Root("regions"), account.Regions)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *resourceAWSAccount) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var planData resourceAWSAccountModel
	diags := req.Plan.Get(ctx, &planData)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	accountData := cloudproviderapi.AWSAccount{}
	accountData.RoleARN = planData.RoleARN.ValueString()
	accountData.Name = planData.Name.ValueString()
	diags = planData.Regions.ElementsAs(ctx, &accountData.Regions, false)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	account, err := r.client.UpdateAWSAccount(
		ctx,
		planData.StackID.ValueString(),
		planData.ResourceID.ValueString(),
		accountData,
	)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update AWS Account", err.Error())
		return
	}

	diags = resp.State.SetAttribute(ctx, path.Root("role_arn"), account.RoleARN)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	diags = resp.State.SetAttribute(ctx, path.Root("name"), account.Name)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	diags = resp.State.SetAttribute(ctx, path.Root("regions"), account.Regions)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r resourceAWSAccount) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data resourceAWSAccountModel
	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteAWSAccount(
		ctx,
		data.StackID.ValueString(),
		data.ResourceID.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete AWS Account", err.Error())
		return
	}

	resp.State.Set(ctx, nil)
}
