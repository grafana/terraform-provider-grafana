package cloudprovider

import (
	"context"

	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common/cloudproviderapi"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	datasourceAWSCloudWatchScrapeJobsTerraformName = "grafana_cloud_provider_aws_cloudwatch_scrape_jobs"
	datasourceAWSCloudWatchScrapeJobsTerraformID   = common.NewResourceID(common.StringIDField("stack_id"))
)

type datasourceAWSCloudWatchScrapeJobsModel struct {
	ID         types.String                              `tfsdk:"id"`
	StackID    types.String                              `tfsdk:"stack_id"`
	ScrapeJobs []awsCloudWatchScrapeJobTFDataSourceModel `tfsdk:"scrape_job"`
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

func (r datasourceAWSCloudWatchScrapeJobs) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = datasourceAWSCloudWatchScrapeJobsTerraformName
}

func (r datasourceAWSCloudWatchScrapeJobs) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
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
					Attributes: func() map[string]schema.Attribute {
						attrs := make(map[string]schema.Attribute, len(datasourceAWSCloudWatchScrapeJobTerraformSchema.Attributes))
						for k, v := range datasourceAWSCloudWatchScrapeJobTerraformSchema.Attributes {
							attrs[k] = v
						}
						attrs["stack_id"] = schema.StringAttribute{
							Description: "The Stack ID of the Grafana Cloud instance. Part of the Terraform Resource ID.",
							Computed:    true,
						}
						attrs["name"] = schema.StringAttribute{
							Description: "The name of the AWS CloudWatch Scrape Job. Part of the Terraform Resource ID.",
							Computed:    true,
						}
						return attrs
					}(),
					Blocks: datasourceAWSCloudWatchScrapeJobTerraformSchema.Blocks,
				},
			},
		},
	}
}

func (r datasourceAWSCloudWatchScrapeJobs) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data datasourceAWSCloudWatchScrapeJobsModel
	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	jobs, err := r.client.ListAWSCloudWatchScrapeJobs(
		ctx,
		data.StackID.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError("Failed to list AWS CloudWatch Scrape Jobs", err.Error())
		return
	}

	scrapeJobs := make([]awsCloudWatchScrapeJobTFDataSourceModel, len(jobs))
	for i, jobResp := range jobs {
		jobTF, diags := generateAWSCloudWatchScrapeJobDataSourceTFModel(ctx, data.StackID.ValueString(), jobResp)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		scrapeJobs[i] = jobTF
	}

	resp.State.Set(ctx, &datasourceAWSCloudWatchScrapeJobsModel{
		ID:         types.StringValue(datasourceAWSCloudWatchScrapeJobsTerraformID.Make(data.StackID.ValueString())),
		StackID:    data.StackID,
		ScrapeJobs: scrapeJobs,
	})
}
