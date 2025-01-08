package cloud

import (
	"context"

	"github.com/grafana/terraform-provider-grafana/v3/internal/common"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var dataSourcePrivateDataSourceConnectNetworksName = "grafana_cloud_private_data_source_connect_networks"

func datasourcePrivateDataSourceConnectNetworks() *common.DataSource {
	return common.NewDataSource(
		common.CategoryCloud,
		dataSourcePrivateDataSourceConnectNetworksName,
		&PDCNetworksDataSource{},
	)
}

type PDCNetworksDataSource struct {
	basePluginFrameworkDataSource
}

func (r *PDCNetworksDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = dataSourcePrivateDataSourceConnectNetworksName
}

func (r *PDCNetworksDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `
Fetches Private Data source Connect networks from Grafana Cloud.

* [Official documentation](https://grafana.com/docs/grafana-cloud/account-management/authentication-and-permissions/access-policies/)
* [API documentation](https://grafana.com/docs/grafana-cloud/developer-resources/api-reference/cloud-api/#list-access-policies)

Required access policy scopes:

* accesspolicies:read`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The ID of this datasource. This is an internal identifier used by the provider to track this datasource.",
			},
			"region_filter": schema.StringAttribute{
				Optional:    true,
				Description: "If set, only private data source connect networks in the specified region will be returned. This is faster than filtering in Terraform.",
			},
			"name_filter": schema.StringAttribute{
				Optional:    true,
				Description: "If set, only private data source connect networks with the specified name will be returned. This is faster than filtering in Terraform.",
			},
			"private_data_source_connect_networks": schema.SetAttribute{
				Computed: true,
				ElementType: types.ObjectType{
					AttrTypes: map[string]attr.Type{
						"id":           types.StringType,
						"region":       types.StringType,
						"name":         types.StringType,
						"display_name": types.StringType,
						"status":       types.StringType,
					},
				},
			},
		},
	}
}

type PDCNetworksDataSourcePolicyModel struct {
	ID          types.String `tfsdk:"id"`
	Region      types.String `tfsdk:"region"`
	Name        types.String `tfsdk:"name"`
	DisplayName types.String `tfsdk:"display_name"`
	Status      types.String `tfsdk:"status"`
}

type PDCNetworksDataSourceModel struct {
	ID                        types.String                       `tfsdk:"id"`
	NameFilter                types.String                       `tfsdk:"name_filter"`
	RegionFilter              types.String                       `tfsdk:"region_filter"`
	PrivateDataSourceNetworks []PDCNetworksDataSourcePolicyModel `tfsdk:"private_data_source_networks"`
}

func (r *PDCNetworksDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	// Read Terraform state data into the model
	var data PDCNetworksDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	var regions []string
	if data.RegionFilter.ValueString() != "" {
		regions = append(regions, data.RegionFilter.ValueString())
	} else {
		apiResp, _, err := r.client.StackRegionsAPI.GetStackRegions(ctx).Execute()
		if err != nil {
			resp.Diagnostics = diag.Diagnostics{diag.NewErrorDiagnostic("Failed to get stack regions", err.Error())}
			return
		}
		for _, region := range apiResp.Items {
			regions = append(regions, region.FormattedApiStackRegionAnyOf.Slug)
		}
	}

	data.PrivateDataSourceNetworks = []PDCNetworksDataSourcePolicyModel{}
	for _, region := range regions {
		apiResp, _, err := r.client.AccesspoliciesAPI.GetAccessPolicies(ctx).Region(region).Execute()
		if err != nil {
			resp.Diagnostics = diag.Diagnostics{diag.NewErrorDiagnostic("Failed to get access policies", err.Error())}
			return
		}
		for _, policy := range apiResp.Items {
			if data.NameFilter.ValueString() != "" && data.NameFilter.ValueString() != policy.Name {
				continue
			}
			data.PrivateDataSourceNetworks = append(data.PrivateDataSourceNetworks, PDCNetworksDataSourcePolicyModel{
				ID:          types.StringValue(*policy.Id),
				Region:      types.StringValue(region),
				Name:        types.StringValue(policy.Name),
				DisplayName: types.StringValue(*policy.DisplayName),
				Status:      types.StringValue(*policy.Status),
			})
		}
	}
	data.ID = types.StringValue(data.RegionFilter.ValueString() + "-" + data.NameFilter.ValueString()) // Unique ID

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}
