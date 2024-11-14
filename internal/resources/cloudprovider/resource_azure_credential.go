package cloudprovider

import (
	"context"

	"github.com/grafana/terraform-provider-grafana/v3/internal/common"
	"github.com/grafana/terraform-provider-grafana/v3/internal/common/cloudproviderapi"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	//nolint:gosec
	resourceAzureCredentialTerraformName = "grafana_cloud_provider_azure_credential"
	resourceAzureCredentialTerraformID   = common.NewResourceID(common.StringIDField("stack_id"), common.StringIDField("resource_id"))
)

type resourceAzureCredentialModel struct {
	ID           types.String `tfsdk:"id"`
	Name         types.String `tfsdk:"name"`
	TenantID     types.String `tfsdk:"tenant_id"`
	ClientID     types.String `tfsdk:"client_id"`
	StackID      types.String `tfsdk:"stack_id"`
	ClientSecret types.String `tfsdk:"client_secret"`
	ResourceID   types.String `tfsdk:"resource_id"`
}

type resourceAzureCredential struct {
	client *cloudproviderapi.Client
}

func makeResourceAzureCredential() *common.Resource {
	return common.NewResource(
		common.CategoryCloudProvider,
		resourceAzureCredentialTerraformName,
		resourceAzureCredentialTerraformID,
		&resourceAzureCredential{},
	)
}

func (r *resourceAzureCredential) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *resourceAzureCredential) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = resourceAzureCredentialTerraformName
}

func (r *resourceAzureCredential) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
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
				Description: "The name of the Azure Credential.",
				Required:    true,
			},
			"client_id": schema.StringAttribute{
				Description: "The client ID of the Azure Credential.",
				Required:    true,
			},
			"tenant_id": schema.StringAttribute{
				Description: "The tenant ID of the Azure Credential.",
				Required:    true,
			},
			"client_secret": schema.StringAttribute{
				Description: "The client secret of the Azure Credential.",
				Required:    true,
				Sensitive:   true,
			},
		},
	}
}

func (r *resourceAzureCredential) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data resourceAzureCredentialModel
	diags := req.Plan.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	azureCredential := cloudproviderapi.AzureCredential{
		Name:         data.Name.ValueString(),
		TenantID:     data.TenantID.ValueString(),
		ClientID:     data.ClientID.ValueString(),
		ClientSecret: data.ClientSecret.ValueString(),
	}

	credential, err := r.client.CreateAzureCredential(
		ctx,
		data.StackID.ValueString(),
		azureCredential,
	)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create Azure credential", err.Error())
		return
	}

	resp.State.Set(ctx, &resourceAzureCredentialModel{
		ID:           types.StringValue(resourceAzureCredentialTerraformID.Make(data.StackID.ValueString(), credential.ID)),
		Name:         data.Name,
		TenantID:     data.TenantID,
		ClientID:     data.ClientID,
		StackID:      data.StackID,
		ClientSecret: data.ClientSecret,
		ResourceID:   types.StringValue(credential.ID),
	})
}

func (r *resourceAzureCredential) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data resourceAzureCredentialModel
	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	credential, err := r.client.GetAzureCredential(
		ctx,
		data.StackID.ValueString(),
		data.ResourceID.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError("Failed to get Azure credential", err.Error())
		return
	}

	diags = resp.State.SetAttribute(ctx, path.Root("name"), credential.Name)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.SetAttribute(ctx, path.Root("tenant_id"), credential.TenantID)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.SetAttribute(ctx, path.Root("client_id"), credential.ClientID)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *resourceAzureCredential) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var planData resourceAzureCredentialModel
	diags := req.Plan.Get(ctx, &planData)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	credential := cloudproviderapi.AzureCredential{}
	credential.Name = planData.Name.ValueString()
	credential.TenantID = planData.TenantID.ValueString()
	credential.ClientID = planData.ClientID.ValueString()
	credential.ClientSecret = planData.ClientSecret.ValueString()

	credentialResponse, err := r.client.UpdateAzureCredential(
		ctx,
		planData.StackID.ValueString(),
		planData.ResourceID.ValueString(),
		credential,
	)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update Azure credential", err.Error())
		return
	}

	diags = resp.State.SetAttribute(ctx, path.Root("name"), credentialResponse.Name)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.SetAttribute(ctx, path.Root("tenant_id"), credentialResponse.TenantID)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.SetAttribute(ctx, path.Root("client_id"), credentialResponse.ClientID)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.SetAttribute(ctx, path.Root("client_secret"), planData.ClientSecret)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *resourceAzureCredential) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data resourceAzureCredentialModel
	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteAzureCredential(
		ctx,
		data.StackID.ValueString(),
		data.ResourceID.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete Azure credential", err.Error())
		return
	}

	resp.State.Set(ctx, nil)
}
