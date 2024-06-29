package cloudprovider

import (
	"context"
	"fmt"
	"strings"

	"github.com/grafana/terraform-provider-grafana/v3/internal/common"
	"github.com/grafana/terraform-provider-grafana/v3/internal/common/cloudproviderapi"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	resourceAWSAccountTerraformName = "grafana_cloud_provider_aws_account"
	resourceAWSAccountTerraformID   = common.NewResourceID(common.StringIDField("stack_id"), common.StringIDField("resource_id"))
)

type resourceAWSAccount struct {
	client     *cloudproviderapi.Client
	ID         types.String `tfsdk:"id"`
	StackID    types.String `tfsdk:"stack_id"`
	ResourceID types.String `tfsdk:"resource_id"`
	RoleARN    types.String `tfsdk:"role_arn"`
	Regions    types.Set    `tfsdk:"regions"`
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

	client, err := withClient(ctx, req, resp)
	if err != nil {
		return
	}

	r.client = client
}

func (r *resourceAWSAccount) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = resourceAWSAccountTerraformName
}

func (r *resourceAWSAccount) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The Terraform Resource ID. This has the format \"{{ stack_id }}:{{ resource_id }}\".",
				Computed:    true,
			},
			"stack_id": schema.StringAttribute{
				Description: "The StackID of the Grafana Cloud instance. Part of the Terraform Resource ID.",
				Required:    true,
			},
			"resource_id": schema.StringAttribute{
				Description: "The ID given by the Grafana Cloud Provider API to this AWS Account resource.",
				Computed:    true,
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

func (r *resourceAWSAccount) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(d.Id(), ":", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return nil, fmt.Errorf("invalid import ID: %s", d.Id())
	}
	d.Set("stack_id", parts[0])
	d.Set("resource_id", parts[1])
	return []*schema.ResourceData{d}, nil
}

func (r *resourceAWSAccount) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data resourceAWSAccount
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	accountData := cloudproviderapi.AWSAccount{}
	accountData.RoleARN = data.RoleARN.ValueString()
	diags := data.Regions.ElementsAs(ctx, accountData.Regions, false)
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
	resp.State.Set(ctx, &resourceAWSAccount{
		ID:         types.StringValue(fmt.Sprintf("%s:%s", data.StackID, account.ID)),
		StackID:    data.StackID,
		ResourceID: types.StringValue(account.ID),
		RoleARN:    data.RoleARN,
		Regions:    data.Regions,
	})
}

func (r *resourceAWSAccount) Read(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data resourceAWSAccount
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	accountData := cloudproviderapi.AWSAccount{}
	accountData.RoleARN = data.RoleARN.ValueString()
	diags := data.Regions.ElementsAs(ctx, accountData.Regions, false)
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
	resp.State.Set(ctx, &resourceAWSAccount{
		ID:         types.StringValue(fmt.Sprintf("%s:%s", data.StackID, account.ID)),
		StackID:    data.StackID,
		ResourceID: types.StringValue(account.ID),
		RoleARN:    data.RoleARN,
		Regions:    data.Regions,
	})
}

func (r *resourceAWSAccount) Update(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data resourceAWSAccount
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	accountData := cloudproviderapi.AWSAccount{}
	accountData.RoleARN = data.RoleARN.ValueString()
	diags := data.Regions.ElementsAs(ctx, accountData.Regions, false)
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
	resp.State.Set(ctx, &resourceAWSAccount{
		ID:         types.StringValue(fmt.Sprintf("%s:%s", data.StackID, account.ID)),
		StackID:    data.StackID,
		ResourceID: types.StringValue(account.ID),
		RoleARN:    data.RoleARN,
		Regions:    data.Regions,
	})
}

func (r *resourceAWSAccount) Delete(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data resourceAWSAccount
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	accountData := cloudproviderapi.AWSAccount{}
	accountData.RoleARN = data.RoleARN.ValueString()
	diags := data.Regions.ElementsAs(ctx, accountData.Regions, false)
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
	resp.State.Set(ctx, &resourceAWSAccount{
		ID:         types.StringValue(fmt.Sprintf("%s:%s", data.StackID, account.ID)),
		StackID:    data.StackID,
		ResourceID: types.StringValue(account.ID),
		RoleARN:    data.RoleARN,
		Regions:    data.Regions,
	})
}
