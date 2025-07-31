package connections

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"

	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common/connectionsapi"
)

type datasourceMetricsEndpointScrapeJob struct {
	client *connectionsapi.Client
}

func makeDatasourceMetricsEndpointScrapeJob() *common.DataSource {
	return common.NewDataSource(
		common.CategoryConnections,
		resourceMetricsEndpointScrapeJobTerraformName,
		&datasourceMetricsEndpointScrapeJob{},
	)
}

func (r *datasourceMetricsEndpointScrapeJob) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (r *datasourceMetricsEndpointScrapeJob) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = resourceMetricsEndpointScrapeJobTerraformName
}

func (r *datasourceMetricsEndpointScrapeJob) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The Terraform Resource ID. This has the format \"{{ stack_id }}:{{ name }}\".",
				Computed:    true,
			},
			"stack_id": schema.StringAttribute{
				Description: "The Stack ID of the Grafana Cloud instance. Part of the Terraform Resource ID.",
				Required:    true,
			},
			"name": schema.StringAttribute{
				Description: "The name of the Metrics Endpoint Scrape Job. Part of the Terraform Resource ID.",
				Required:    true,
			},
			"enabled": schema.BoolAttribute{
				Description: "Whether the metrics endpoint scrape job is enabled or not.",
				Computed:    true,
			},
			"authentication_method": schema.StringAttribute{
				Description: "Method to pass authentication credentials: basic or bearer.",
				Computed:    true,
			},
			"authentication_bearer_token": schema.StringAttribute{
				Description: "Token for authentication bearer.",
				Sensitive:   true,
				Computed:    true,
			},
			"authentication_basic_username": schema.StringAttribute{
				Description: "Username for basic authentication.",
				Computed:    true,
			},
			"authentication_basic_password": schema.StringAttribute{
				Description: "Password for basic authentication.",
				Sensitive:   true,
				Computed:    true,
			},
			"url": schema.StringAttribute{
				Description: "The url to scrape metrics.",
				Computed:    true,
			},
			"scrape_interval_seconds": schema.Int64Attribute{
				Description: "Frequency for scraping the metrics endpoint: 30, 60, or 120 seconds.",
				Computed:    true,
			},
		},
	}
}

func (r *datasourceMetricsEndpointScrapeJob) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var dataTF metricsEndpointScrapeJobTFModel
	diags := req.Config.Get(ctx, &dataTF)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	jobClientModel, err := r.client.GetMetricsEndpointScrapeJob(
		ctx,
		dataTF.StackID.ValueString(),
		dataTF.Name.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError("failed to get metrics endpoint scrape job", err.Error())
		return
	}

	resp.State.Set(ctx, convertClientModelToTFModel(dataTF.StackID.ValueString(), dataTF.Name.ValueString(), jobClientModel))
}
