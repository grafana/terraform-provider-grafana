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
	datasourceAWSCloudWatchScrapeJobsTerraformID   = common.NewResourceID(common.StringIDField("stack_id"))
)

type datasourceAWSCloudWatchScrapeJobsModel struct {
	ID         types.String            `tfsdk:"id"`
	StackID    types.String            `tfsdk:"stack_id"`
	ScrapeJobs []awsCWScrapeJobTFModel `tfsdk:"scrape_job"`
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
			"id": schema.StringAttribute{
				Description: "The Terraform Resource ID. This has the format \"{{ stack_id }}\".",
				Computed:    true,
			},
			"stack_id": schema.StringAttribute{
				Description: "The Stack ID of the Grafana Cloud instance. Part of the Terraform Resource ID.",
				Required:    true,
			},
		},
		Blocks: map[string]schema.Block{
			"scrape_job": schema.ListNestedBlock{
				Description: "A list of AWS CloudWatch Scrape Job objects associated with the given StackID.",
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
	scrapeJobs := make([]awsCWScrapeJobTFModel, len(scrapeJobsResp))
	for i, scrapeJobData := range scrapeJobsResp {
		scrapeJob, diags := convertScrapeJobClientModelToTFModel(ctx, scrapeJobData)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		scrapeJobs[i] = *scrapeJob
	}

	resp.State.Set(ctx, &datasourceAWSCloudWatchScrapeJobsModel{
		ID:         types.StringValue(datasourceAWSCloudWatchScrapeJobsTerraformID.Make(data.StackID.ValueString())),
		StackID:    data.StackID,
		ScrapeJobs: scrapeJobs,
	})
}
