package cloud

import (
	"context"
	"strconv"

	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &CloudOrganizationDataSource{}
var _ datasource.DataSourceWithConfigure = &CloudOrganizationDataSource{}

var dataSourceCloudOrganizationName = "grafana_cloud_organization"

func datasourceOrganization() *common.DataSource {
	return common.NewDataSource(
		common.CategoryCloud,
		dataSourceCloudOrganizationName,
		&CloudOrganizationDataSource{},
	)
}

type CloudOrganizationDataSource struct {
	basePluginFrameworkDataSource
}

func (r *CloudOrganizationDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = dataSourceCloudOrganizationName
}

func (r *CloudOrganizationDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `
Fetches a Grafana Cloud organization.

* [Official documentation](https://grafana.com/docs/grafana-cloud/)
* [API documentation](https://grafana.com/docs/grafana-cloud/developer-resources/api-reference/cloud-api/#organizations)

Required access policy scopes:

* orgs:read`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The organization ID.",
			},
			"slug": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The organization slug.",
			},
			"name": schema.StringAttribute{
				Computed:    true,
				Description: "The organization name.",
			},
			"url": schema.StringAttribute{
				Computed:    true,
				Description: "The organization URL.",
			},
			"created_at": schema.StringAttribute{
				Computed:    true,
				Description: "The date and time the organization was created.",
			},
			"updated_at": schema.StringAttribute{
				Computed:    true,
				Description: "The date and time the organization was last updated.",
			},
		},
	}
}

type CloudOrganizationDataSourceModel struct {
	ID        types.String `tfsdk:"id"`
	Slug      types.String `tfsdk:"slug"`
	Name      types.String `tfsdk:"name"`
	URL       types.String `tfsdk:"url"`
	CreatedAt types.String `tfsdk:"created_at"`
	UpdatedAt types.String `tfsdk:"updated_at"`
}

func (r *CloudOrganizationDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	// Read Terraform configuration data into the model
	var data CloudOrganizationDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Determine which identifier to use (id or slug)
	identifier := data.ID.ValueString()
	if identifier == "" {
		identifier = data.Slug.ValueString()
	}

	// Fetch organization from API
	org, _, err := r.client.OrgsAPI.GetOrg(ctx, identifier).Execute()
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to get organization",
			"Could not read organization: "+err.Error(),
		)
		return
	}

	// Map response to model
	id := strconv.FormatInt(int64(org.Id), 10)
	data.ID = types.StringValue(id)
	data.Name = types.StringValue(org.Name)
	data.Slug = types.StringValue(org.Slug)
	data.URL = types.StringValue(org.Url)
	data.CreatedAt = types.StringValue(org.CreatedAt)

	if updatedAt := org.UpdatedAt.Get(); updatedAt != nil {
		data.UpdatedAt = types.StringValue(*updatedAt)
	} else {
		data.UpdatedAt = types.StringNull()
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}
