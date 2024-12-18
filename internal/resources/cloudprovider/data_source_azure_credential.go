package cloudprovider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"

	"github.com/grafana/terraform-provider-grafana/v3/internal/common"
	"github.com/grafana/terraform-provider-grafana/v3/internal/common/cloudproviderapi"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type datasourceAzureCredential struct {
	client *cloudproviderapi.Client
}

func makeDataSourceAzureCredential() *common.DataSource {
	return common.NewDataSource(
		common.CategoryCloudProvider,
		resourceAzureCredentialTerraformName,
		&datasourceAzureCredential{},
	)
}

func (r *datasourceAzureCredential) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (r *datasourceAzureCredential) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = resourceAzureCredentialTerraformName
}

func (r *datasourceAzureCredential) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
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
				Description: "The ID given by the Grafana Cloud Provider API to this Azure Credential resource.",
				Required:    true,
			},
			"name": schema.StringAttribute{
				Description: "The name of the Azure Credential.",
				Computed:    true,
			},
			"client_id": schema.StringAttribute{
				Description: "The client ID of the Azure Credential.",
				Computed:    true,
			},
			"tenant_id": schema.StringAttribute{
				Description: "The tenant ID of the Azure Credential.",
				Computed:    true,
			},
			"client_secret": schema.StringAttribute{
				Description: "The client secret of the Azure Credential.",
				Computed:    true,
				Sensitive:   true,
			},
		},
		Blocks: map[string]schema.Block{
			"resource_discovery_tag_filter": schema.ListNestedBlock{
				Description: "The list of tag filters to apply to resources.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"key": schema.StringAttribute{
							Description: "The key of the tag filter.",
							Computed:    true,
						},
						"value": schema.StringAttribute{
							Description: "The value of the tag filter.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (r *datasourceAzureCredential) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data resourceAzureCredentialModel
	diags := req.Config.Get(ctx, &data)
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
		resp.Diagnostics.AddError("Failed to get Azure Credential", err.Error())
		return
	}

	diags = resp.State.SetAttribute(ctx, path.Root("id"), types.StringValue(resourceAzureCredentialTerraformID.Make(data.StackID.ValueString(), data.ResourceID.ValueString())))
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.SetAttribute(ctx, path.Root("name"), credential.Name)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.SetAttribute(ctx, path.Root("client_id"), credential.ClientID)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.SetAttribute(ctx, path.Root("tenant_id"), credential.TenantID)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.SetAttribute(ctx, path.Root("client_secret"), credential.ClientSecret)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	convertedTagFilters, diags := r.convertTagFilters(ctx, credential.ResourceTagFilters)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	diags = resp.State.SetAttribute(ctx, path.Root("resource_discovery_tag_filter"), convertedTagFilters)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *datasourceAzureCredential) convertTagFilters(ctx context.Context, apiTagFilters []cloudproviderapi.TagFilter) (types.List, diag.Diagnostics) {
	tagFiltersTF := make([]TagFilter, len(apiTagFilters))
	conversionDiags := diag.Diagnostics{}
	tagFilterListObjType := types.ObjectType{AttrTypes: TagFilter{}.attrTypes()}

	for i, apiTagFilter := range apiTagFilters {
		tagFiltersTF[i] = TagFilter{
			Key:   types.StringValue(apiTagFilter.Key),
			Value: types.StringValue(apiTagFilter.Value),
		}
	}

	tagFiltersTFList, diags := types.ListValueFrom(ctx, tagFilterListObjType, tagFiltersTF)
	conversionDiags.Append(diags...)
	if conversionDiags.HasError() {
		return types.ListNull(tagFilterListObjType), conversionDiags
	}
	return tagFiltersTFList, conversionDiags
}
