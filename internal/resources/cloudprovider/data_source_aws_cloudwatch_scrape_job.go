package cloudprovider

import (
	"context"

	"github.com/grafana/terraform-provider-grafana/v3/internal/common"
	"github.com/grafana/terraform-provider-grafana/v3/internal/common/cloudproviderapi"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

var (
	datasourceAWSCloudWatchScrapeJobTerraformSchema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The Terraform Resource ID. This has the format \"{{ stack_id }}:{{ job_name }}\".",
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
				Description: "The ID assigned by the Grafana Cloud Provider API to an AWS Account resource that should be associated with this CloudWatch Scrape Job.",
				Computed:    true,
			},
			"regions": schema.SetAttribute{
				Description: "A set of AWS region names that this CloudWatch Scrape Job applies to.",
				Computed:    true,
				ElementType: types.StringType,
			},
		},
		Blocks: map[string]schema.Block{
			"service_configuration": schema.SetNestedBlock{
				Description: "Each block dictates what this CloudWatch Scrape Job should scrape for the specified AWS service.",
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
						"is_custom_namespace": schema.BoolAttribute{
							Description: "Whether the service name is a custom, user-generated metrics namespace, as opposed to a standard AWS service metrics namespace.",
							Computed:    true,
						},
					},
					Blocks: map[string]schema.Block{
						"metric": schema.SetNestedBlock{
							Description: "Each block configures a metric and their statistics to scrape.",
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
						"resource_discovery_tag_filter": schema.SetNestedBlock{
							Description: "Each block configures a tag filter applied to discovery of resource entities in the associated AWS account.",
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
	converted, diags := scrapeJobClientModelToTerraformModel(ctx, TestAWSCloudWatchScrapeJobData)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.State.Set(ctx, converted)
}

func scrapeJobClientModelToTerraformModel(ctx context.Context, scrapeJobData cloudproviderapi.AWSCloudWatchScrapeJob) (*awsCloudWatchScrapeJobModel, diag.Diagnostics) {
	conversionDiags := diag.Diagnostics{}
	converted := &awsCloudWatchScrapeJobModel{
		ID:                   types.StringValue(resourceAWSCloudWatchScrapeJobTerraformID.Make(scrapeJobData.StackID, scrapeJobData.Name)),
		StackID:              types.StringValue(scrapeJobData.StackID),
		Name:                 types.StringValue(scrapeJobData.Name),
		Enabled:              types.BoolValue(scrapeJobData.Enabled),
		AWSAccountResourceID: types.StringValue(scrapeJobData.AWSAccountResourceID),
	}

	regions, diags := types.SetValueFrom(ctx, basetypes.StringType{}, &scrapeJobData.Regions)
	diags.Append(diags...)
	if diags.HasError() {
		return nil, conversionDiags
	}
	converted.Regions = regions

	for _, serviceConfigData := range scrapeJobData.ServiceConfigurations {
		serviceConfig := awsCloudWatchScrapeJobServiceConfigurationModel{
			Name:                  types.StringValue(serviceConfigData.Name),
			ScrapeIntervalSeconds: types.Int64Value(serviceConfigData.ScrapeIntervalSeconds),
			IsCustomNamespace:     types.BoolValue(serviceConfigData.IsCustomNamespace),
		}

		metricsData := make([]awsCloudWatchScrapeJobMetricModel, len(serviceConfigData.Metrics))
		for i, metricData := range serviceConfigData.Metrics {
			metricsData[i] = awsCloudWatchScrapeJobMetricModel{
				Name: types.StringValue(metricData.Name),
			}
			statistics, diags := types.SetValueFrom(ctx, basetypes.StringType{}, &metricData.Statistics)
			conversionDiags.Append(diags...)
			if conversionDiags.HasError() {
				return nil, conversionDiags
			}
			metricsData[i].Statistics = statistics
		}
		serviceConfig.Metrics = metricsData

		tagFiltersData := make([]awsCloudWatchScrapeJobTagFilterModel, len(serviceConfigData.ResourceDiscoveryTagFilters))
		for i, tagFilterData := range serviceConfigData.ResourceDiscoveryTagFilters {
			tagFiltersData[i] = awsCloudWatchScrapeJobTagFilterModel{
				Key:   types.StringValue(tagFilterData.Key),
				Value: types.StringValue(tagFilterData.Value),
			}
		}
		serviceConfig.ResourceDiscoveryTagFilters = tagFiltersData

		tagsToAdd, diags := types.SetValueFrom(ctx, basetypes.StringType{}, &serviceConfigData.TagsToAddToMetrics)
		conversionDiags.Append(diags...)
		if conversionDiags.HasError() {
			return nil, conversionDiags
		}
		serviceConfig.TagsToAddToMetrics = tagsToAdd

		converted.ServiceConfigurationBlocks = append(converted.ServiceConfigurationBlocks, serviceConfig)
	}

	return converted, conversionDiags
}
