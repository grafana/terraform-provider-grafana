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
	// It is intended to use the same schema for scrape jobs between the singular and
	// plural versions of the data source.
	datasourceAWSCloudWatchScrapeJobTerraformSchema = schema.Schema{
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
				Description: "The name of the CloudWatch Scrape Job. Part of the Terraform Resource ID.",
				Required:    true,
			},
			"enabled": schema.BoolAttribute{
				Description: "Whether the CloudWatch Scrape Job is enabled or not.",
				Computed:    true,
			},
			"aws_account_resource_id": schema.StringAttribute{
				Description: "The ID assigned by the Grafana Cloud Provider API to an AWS Account resource that should be associated with this CloudWatch Scrape Job. This can be provided by the `resource_id` attribute of the `grafana_cloud_provider_aws_account` resource.",
				Computed:    true,
			},
			"role_arn": schema.StringAttribute{
				Description: "The AWS ARN of the IAM role associated with the AWS Account resource that is being used by this CloudWatch Scrape Job.",
				Computed:    true,
			},
			"regions": schema.SetAttribute{
				Description: "The set of AWS region names that this CloudWatch Scrape Job is configured to scrape.",
				Computed:    true,
				ElementType: types.StringType,
			},
			"regions_subset_override_used": schema.BoolAttribute{
				Description: "When true, the `regions` attribute will be the set of regions configured in the override. When false, the `regions` attribute will be the set of regions belonging to the AWS Account resource that is associated with this CloudWatch Scrape Job.",
				Computed:    true,
			},
			"export_tags": schema.BoolAttribute{
				Description: "When enabled, AWS resource tags are exported as Prometheus labels to metrics formatted as `aws_<service_name>_info`.",
				Computed:    true,
			},
			"disabled_reason": schema.StringAttribute{
				Description: "When the CloudWatch Scrape Job is disabled, this will show the reason that it is in that state.",
				Computed:    true,
			},
		},
		Blocks: map[string]schema.Block{
			"service": schema.ListNestedBlock{
				Description: "One or more configuration blocks to dictate what this CloudWatch Scrape Job should scrape. Each block must have a distinct `name` attribute. When accessing this as an attribute reference, it is a list of objects.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Description: "The name of the service to scrape. See https://grafana.com/docs/grafana-cloud/monitor-infrastructure/aws/cloudwatch-metrics/services/ for supported services, metrics, and their statistics.",
							Computed:    true,
						},
						"scrape_interval_seconds": schema.Int64Attribute{
							Description: "The interval in seconds to scrape the service. See https://grafana.com/docs/grafana-cloud/monitor-infrastructure/aws/cloudwatch-metrics/services/ for supported scrape intervals.",
							Computed:    true,
						},
						"tags_to_add_to_metrics": schema.SetAttribute{
							Description: "A set of tags to add to all metrics exported by this scrape job, for use in PromQL queries.",
							Computed:    true,
							ElementType: types.StringType,
						},
					},
					Blocks: map[string]schema.Block{
						"metric": schema.ListNestedBlock{
							Description: "One or more configuration blocks to configure metrics and their statistics to scrape. Each block must represent a distinct metric name. When accessing this as an attribute reference, it is a list of objects.",
							NestedObject: schema.NestedBlockObject{
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
						"resource_discovery_tag_filter": schema.ListNestedBlock{
							Description: "One or more configuration blocks to configure tag filters applied to discovery of resource entities in the associated AWS account. When accessing this as an attribute reference, it is a list of objects.",
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
				},
			},
			"custom_namespace": schema.ListNestedBlock{
				Description: "Zero or more configuration blocks to configure custom namespaces for the CloudWatch Scrape Job to scrape. Each block must have a distinct `name` attribute. When accessing this as an attribute reference, it is a list of objects.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Description: "The name of the custom namespace to scrape.",
							Computed:    true,
						},
						"scrape_interval_seconds": schema.Int64Attribute{
							Description: "The interval in seconds to scrape the custom namespace.",
							Computed:    true,
						},
					},
					Blocks: map[string]schema.Block{
						"metric": schema.ListNestedBlock{
							Description: "One or more configuration blocks to configure metrics and their statistics to scrape. Each block must represent a distinct metric name. When accessing this as an attribute reference, it is a list of objects.",
							NestedObject: schema.NestedBlockObject{
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
					},
				},
			},
			"static_label": schema.ListNestedBlock{
				Description: "Zero or more configuration blocks to configure static labels to add to all metrics exported by this scrape job.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"label": schema.StringAttribute{
							Description: "The label.",
							Computed:    true,
						},
						"value": schema.StringAttribute{
							Description: "The value of the label.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
)

type datasourceAWSCloudWatchScrapeJob struct {
	client *cloudproviderapi.Client
}

func makeDatasourceAWSCloudWatchScrapeJob() *common.DataSource {
	return common.NewDataSource(
		common.CategoryCloudProvider,
		resourceAWSCloudWatchScrapeJobTerraformName,
		&datasourceAWSCloudWatchScrapeJob{},
	)
}

func (r *datasourceAWSCloudWatchScrapeJob) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (r *datasourceAWSCloudWatchScrapeJob) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = resourceAWSCloudWatchScrapeJobTerraformName
}

func (r *datasourceAWSCloudWatchScrapeJob) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = datasourceAWSCloudWatchScrapeJobTerraformSchema
}

func (r *datasourceAWSCloudWatchScrapeJob) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data awsCWScrapeJobTFDataSourceModel
	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	jobResp, err := r.client.GetAWSCloudWatchScrapeJob(
		ctx,
		data.StackID.ValueString(),
		data.Name.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError("Failed to get AWS CloudWatch scrape job", err.Error())
		return
	}

	jobTF, diags := generateCloudWatchScrapeJobDataSourceTFModel(ctx, data.StackID.ValueString(), jobResp)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.State.Set(ctx, jobTF)
}
