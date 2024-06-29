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
	datasourceAWSCWScrapeJobsTerraformName = "grafana_cloud_provider_aws_cloudwatch_scrape_jobs"
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
		datasourceAWSCWScrapeJobsTerraformName,
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
	resp.TypeName = datasourceAWSCWScrapeJobsTerraformName
}

func (r *datasourceAWSCloudWatchScrapeJobs) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"stack_id": schema.StringAttribute{
				Description: "The Stack ID of the Grafana Cloud instance. Part of the Terraform Resource ID.",
				Required:    true,
			},
			"scrape_jobs": schema.SetNestedAttribute{
				Description: "The set of AWS CloudWatch Scrape Jobs associated with the given StackID.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "The Terraform Resource ID. This has the format \"{{ stack_id }}:{{ job_name }}\".",
							Computed:    true,
						},
						"stack_id": schema.StringAttribute{
							Description: "The Stack ID of the Grafana Cloud instance. Part of the Terraform Resource ID.",
							Computed:    true,
						},
						"name": schema.StringAttribute{
							Description: "The name of the CloudWatch Scrape Job. Part of the Terraform Resource ID.",
							Computed:    true,
						},
						"enabled": schema.BoolAttribute{
							Description: "Whether the CloudWatch Scrape Job is enabled or not.",
							Computed:    true,
						},
						"aws_account_resource_id": schema.StringAttribute{
							Description: "The ID assigned by the Grafana Cloud Provider API to an AWS Account resource that should be associated with this CloudWatch Scrape Job.",
							Computed:    true,
						},
						"regions": schema.SetAttribute{
							Description: "A set of AWS region names that this CloudWatch Scrape Job applies to.",
							Computed:    true,
							ElementType: types.StringType,
						},
						"service_configuration": schema.SetNestedAttribute{
							Description: "Each block is a service configuration that dictates what this CloudWatch Scrape Job should scrape for the specified AWS service.",
							Computed:    true,
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"name": schema.StringAttribute{
										Description: "The name of the service to scrape. See https://grafana.com/docs/grafana-cloud/monitor-infrastructure/aws/cloudwatch-metrics/services/ for supported services, metrics, and their statistics.",
										Computed:    true,
									},
									"metrics": schema.SetNestedAttribute{
										Description: "A set of metrics to scrape.",
										Computed:    true,
										NestedObject: schema.NestedAttributeObject{
											Attributes: map[string]schema.Attribute{
												"name": schema.StringAttribute{
													Description: "The name of the metric to scrape.",
													Computed:    true,
												},
												"statistics": schema.SetAttribute{
													Description: "A set of statistics to scrape.",
													Computed:    true,
													ElementType: types.StringType,
												},
											},
										},
									},
									"scrape_interval_seconds": schema.Int64Attribute{
										Description: "The interval in seconds to scrape the service. See https://grafana.com/docs/grafana-cloud/monitor-infrastructure/aws/cloudwatch-metrics/services/ for supported scrape intervals.",
										Computed:    true,
									},
									"resource_discovery_tag_filters": schema.SetNestedAttribute{
										Description: "A set of tag filters to use for discovery of resource entities in the associated AWS account.",
										Computed:    true,
										NestedObject: schema.NestedAttributeObject{
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
									"tags_to_add_to_metrics": schema.SetAttribute{
										Description: "A set of tags to add to all metrics exported by this scrape job, for use in PromQL queries.",
										Computed:    true,
										ElementType: types.StringType,
									},
									"is_custom_namespace": schema.BoolAttribute{
										Description: "Whether the service name is a custom, user-generated metrics namespace, as opposed to a standard AWS service metrics namespace.",
										Computed:    true,
									},
								},
							},
						},
					},
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
