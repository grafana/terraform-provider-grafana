package cloudprovider

import (
	"context"

	"github.com/grafana/terraform-provider-grafana/v3/internal/common"
	"github.com/grafana/terraform-provider-grafana/v3/internal/common/cloudproviderapi"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type datasourceAWSAccount struct {
	client *cloudproviderapi.Client
}

func makeDataSourceAWSAccount() *common.DataSource {
	return common.NewDataSource(
		common.CategoryCloudProvider,
		resourceAWSAccountTerraformName,
		&datasourceAWSAccount{},
	)
}

func (r *datasourceAWSAccount) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// Configure is called multiple times (sometimes when ProviderData is not yet available), we only want to configure once
	if req.ProviderData == nil || r.client != nil {
		return
	}

	client, err := withClientForDataSource(req, resp)
	if err != nil {
		return
	}

	r.client = client
}

func (r datasourceAWSAccount) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = resourceAWSAccountTerraformName
}

func (r datasourceAWSAccount) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
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
				Required:    true,
			},
			"name": schema.StringAttribute{
				Description: "An optional human-readable name for this AWS Account resource.",
				Computed:    true,
			},
			"role_arn": schema.StringAttribute{
				Description: "An IAM Role ARN string to represent with this AWS Account resource.",
				Computed:    true,
			},
			"regions": schema.SetAttribute{
				Description: "A set of regions that this AWS Account resource applies to.",
				Computed:    true,
				ElementType: types.StringType,
			},
		},
	}
}

func (r datasourceAWSAccount) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data resourceAWSAccountModel
	diags := req.Config.Get(ctx, &data)
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

	diags = resp.State.SetAttribute(ctx, path.Root("id"), types.StringValue(resourceAWSAccountTerraformID.Make(data.StackID.ValueString(), data.ResourceID.ValueString())))
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
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
