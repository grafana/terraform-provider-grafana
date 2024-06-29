package cloudprovider

import (
	"context"

	"github.com/grafana/terraform-provider-grafana/v3/internal/common"
	"github.com/grafana/terraform-provider-grafana/v3/internal/common/cloudproviderapi"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	datasourceAWSCloudWatchScrapeJobsTerraformName = "grafana_cloud_provider_aws_cloudwatch_scrape_jobs"
)

type datasourceAWSCloudWatchScrapeJobsModel struct {
	StackID    types.String `tfsdk:"stack_id"`
	ScrapeJobs types.Set    `tfsdk:"scrape_jobs"`
}

type datasourceAWSCloudWatchScrapeJobs struct {
	client *cloudproviderapi.Client
}

func makeDatasourceAWSCloudWatchScrapeJobs() *common.DataSource {
	return common.NewDataSource(
		common.CategoryCloudProvider,
		datasourceAWSCloudWatchScrapeJobsTerraformName,
		&datasourceAWSCloudWatchScrapeJobs{},
	)
}

func (r *datasourceAWSCloudWatchScrapeJobs) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (r *datasourceAWSCloudWatchScrapeJobs) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = datasourceAWSCloudWatchScrapeJobsTerraformName
}

func (r *datasourceAWSCloudWatchScrapeJobs) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"stack_id": schema.StringAttribute{
				Description: "The Stack ID of the Grafana Cloud instance. Part of the Terraform Resource ID.",
				Required:    true,
			},
		},
		Blocks: map[string]schema.Block{
			"scrape_jobs": schema.SetNestedBlock{
				Description: "The set of AWS CloudWatch Scrape Jobs associated with the given StackID.",
				NestedObject: schema.NestedBlockObject{
					Attributes: datasourceAWSCloudWatchScrapeJobTerraformSchema.Attributes,
					Blocks:     datasourceAWSCloudWatchScrapeJobTerraformSchema.Blocks,
				},
			},
		},
	}
}

func (r *datasourceAWSCloudWatchScrapeJobs) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data datasourceAWSCloudWatchScrapeJobsModel
	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	scrapeJobsResp := []cloudproviderapi.AWSCloudWatchScrapeJob{TestAWSCloudWatchScrapeJobData}
	scrapeJobsData := make([]map[string]interface{}, len(scrapeJobsResp))
	for i, result := range scrapeJobsResp {
		scrapeJobsData[i] = map[string]interface{}{
			"id":                      resourceAWSCloudWatchScrapeJobTerraformID.Make(data.StackID.ValueString(), result.Name),
			"stack_id":                data.StackID.ValueString(),
			"name":                    result.Name,
			"enabled":                 result.Enabled,
			"aws_account_resource_id": result.AWSAccountResourceID,
			"regions":                 result.Regions,
			"service_configurations":  result.ServiceConfigurations,
		}
	}

	scrapeJobs, diags := types.SetValueFrom(ctx, types.SetType{}, &scrapeJobsData)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.State.Set(ctx, &datasourceAWSCloudWatchScrapeJobsModel{
		StackID:    data.StackID,
		ScrapeJobs: scrapeJobs,
	})
}
