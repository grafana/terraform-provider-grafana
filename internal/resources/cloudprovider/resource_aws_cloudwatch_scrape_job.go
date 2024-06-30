package cloudprovider

import (
	"context"
	"fmt"
	"strings"

	"github.com/grafana/terraform-provider-grafana/v3/internal/common"
	"github.com/grafana/terraform-provider-grafana/v3/internal/common/cloudproviderapi"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	resourceAWSCloudWatchScrapeJobTerraformName = "grafana_cloud_provider_aws_cloudwatch_scrape_job"
	resourceAWSCloudWatchScrapeJobTerraformID   = common.NewResourceID(common.StringIDField("stack_id"), common.StringIDField("job_name"))
)

type resourceAWSCloudWatchScrapeJobModel struct {
	ID                   types.String `tfsdk:"id"`
	StackID              types.String `tfsdk:"stack_id"`
	Name                 types.String `tfsdk:"name"`
	Enabled              types.Bool   `tfsdk:"enabled"`
	AWSAccountResourceID types.String `tfsdk:"aws_account_resource_id"`
	Regions              types.Set    `tfsdk:"regions"`
	// TODO(tristan): if the grafana provider is update the Terraform v6 schema,
	// we can consider adding additional support to use Set Nested Attributes, instead of Blocks.
	// See https://developer.hashicorp.com/terraform/plugin/framework/handling-data/attributes#nested-attribute-types
	ServiceConfigurationBlocks []awsCloudWatchScrapeJobServiceConfigurationModel `tfsdk:"service_configuration"`
}
type awsCloudWatchScrapeJobServiceConfigurationModel struct {
	Name                        types.String                           `tfsdk:"name"`
	Metrics                     []awsCloudWatchScrapeJobMetricModel    `tfsdk:"metric"`
	ScrapeIntervalSeconds       types.Int64                            `tfsdk:"scrape_interval_seconds"`
	ResourceDiscoveryTagFilters []awsCloudWatchScrapeJobTagFilterModel `tfsdk:"resource_discovery_tag_filter"`
	TagsToAddToMetrics          types.Set                              `tfsdk:"tags_to_add_to_metrics"`
	IsCustomNamespace           types.Bool                             `tfsdk:"is_custom_namespace"`
}
type awsCloudWatchScrapeJobMetricModel struct {
	Name       types.String `tfsdk:"name"`
	Statistics types.Set    `tfsdk:"statistics"`
}
type awsCloudWatchScrapeJobTagFilterModel struct {
	Key   types.String `tfsdk:"key"`
	Value types.String `tfsdk:"value"`
}

type resourceAWSCloudWatchScrapeJob struct {
	client *cloudproviderapi.Client
}

func makeResourceAWSCloudWatchScrapeJob() *common.Resource {
	return common.NewResource(
		common.CategoryCloudProvider,
		"grafana_cloud_provider_aws_cloudwatch_scrape_job",
		resourceAWSCloudWatchScrapeJobTerraformID,
		&resourceAWSCloudWatchScrapeJob{},
	)
}

func (r *resourceAWSCloudWatchScrapeJob) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Configure is called multiple times (sometimes when ProviderData is not yet available), we only want to configure once
	if req.ProviderData == nil || r.client != nil {
		return
	}

	client, err := withClientForResource(req, resp)
	if err != nil {
		return
	}

	r.client = client
}

func (r *resourceAWSCloudWatchScrapeJob) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = resourceAWSCloudWatchScrapeJobTerraformName
}

func (r *resourceAWSCloudWatchScrapeJob) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
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
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"aws_account_resource_id": schema.StringAttribute{
				Description: "The ID assigned by the Grafana Cloud Provider API to an AWS Account resource that should be associated with this CloudWatch Scrape Job.",
				Required:    true,
			},
			"regions": schema.SetAttribute{
				Description: "A set of AWS region names that this CloudWatch Scrape Job applies to.",
				Required:    true,
				Validators: []validator.Set{
					setvalidator.SizeAtLeast(1),
				},
				ElementType: types.StringType,
			},
		},
		Blocks: map[string]schema.Block{
			"service_configuration": schema.SetNestedBlock{
				Description: "Each block dictates what this CloudWatch Scrape Job should scrape for the specified AWS service.",
				Validators: []validator.Set{
					setvalidator.SizeAtLeast(1),
				},
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Description: "The name of the service to scrape. See https://grafana.com/docs/grafana-cloud/monitor-infrastructure/aws/cloudwatch-metrics/services/ for supported services, metrics, and their statistics.",
							Required:    true,
						},
						"scrape_interval_seconds": schema.Int64Attribute{
							Description: "The interval in seconds to scrape the service. See https://grafana.com/docs/grafana-cloud/monitor-infrastructure/aws/cloudwatch-metrics/services/ for supported scrape intervals.",
							Optional:    true,
							Computed:    true,
							Default:     int64default.StaticInt64(300),
						},
						"tags_to_add_to_metrics": schema.SetAttribute{
							Description: "A set of tags to add to all metrics exported by this scrape job, for use in PromQL queries.",
							Optional:    true,
							ElementType: types.StringType,
						},
						"is_custom_namespace": schema.BoolAttribute{
							Description: "Whether the service name is a custom, user-generated metrics namespace, as opposed to a standard AWS service metrics namespace.",
							Optional:    true,
							Computed:    true,
							Default:     booldefault.StaticBool(false),
						},
					},
					Blocks: map[string]schema.Block{
						"metric": schema.SetNestedBlock{
							Description: "Each block configures a metric and their statistics to scrape.",
							Validators: []validator.Set{
								setvalidator.SizeAtLeast(1),
							},
							NestedObject: schema.NestedBlockObject{
								Attributes: map[string]schema.Attribute{
									"name": schema.StringAttribute{
										Description: "The name of the metric to scrape.",
										Required:    true,
									},
									"statistics": schema.SetAttribute{
										Description: "A set of statistics to scrape.",
										Required:    true,
										Validators: []validator.Set{
											setvalidator.SizeAtLeast(1),
										},
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
										Required:    true,
									},
									"value": schema.StringAttribute{
										Description: "The value of the tag filter.",
										Required:    true,
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

func (r *resourceAWSCloudWatchScrapeJob) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, ":", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Invalid ID: %s", req.ID))
		return
	}
	stackID := parts[0]
	jobName := parts[1]
	// TODO(tristan): use client to get AWS account so we only import a resource that exists
	resp.State.Set(ctx, &resourceAWSCloudWatchScrapeJobModel{
		ID:      types.StringValue(req.ID),
		StackID: types.StringValue(stackID),
		Name:    types.StringValue(jobName),
	})
}

func (r *resourceAWSCloudWatchScrapeJob) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data resourceAWSCloudWatchScrapeJobModel
	diags := req.Plan.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.State.Set(ctx, &resourceAWSCloudWatchScrapeJobModel{
		ID:                         types.StringValue(resourceAWSCloudWatchScrapeJobTerraformID.Make(data.StackID.ValueString(), data.Name.ValueString())),
		StackID:                    data.StackID,
		Name:                       data.Name,
		Enabled:                    data.Enabled,
		AWSAccountResourceID:       data.AWSAccountResourceID,
		Regions:                    data.Regions,
		ServiceConfigurationBlocks: data.ServiceConfigurationBlocks,
	})
}

func (r *resourceAWSCloudWatchScrapeJob) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data resourceAWSCloudWatchScrapeJobModel
	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.State.Set(ctx, &resourceAWSCloudWatchScrapeJobModel{
		ID:                         types.StringValue(resourceAWSCloudWatchScrapeJobTerraformID.Make(data.StackID.ValueString(), data.Name.ValueString())),
		StackID:                    data.StackID,
		Name:                       data.Name,
		Enabled:                    data.Enabled,
		AWSAccountResourceID:       data.AWSAccountResourceID,
		Regions:                    data.Regions,
		ServiceConfigurationBlocks: data.ServiceConfigurationBlocks,
	})
}

func (r *resourceAWSCloudWatchScrapeJob) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var stateData resourceAWSCloudWatchScrapeJobModel
	diags := req.State.Get(ctx, &stateData)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	var configData resourceAWSCloudWatchScrapeJobModel
	diags = req.Config.Get(ctx, &configData)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.State.Set(ctx, &resourceAWSCloudWatchScrapeJobModel{
		ID:                         types.StringValue(resourceAWSCloudWatchScrapeJobTerraformID.Make(stateData.StackID.ValueString(), configData.Name.ValueString())),
		StackID:                    stateData.StackID,
		Name:                       configData.Name,
		Enabled:                    configData.Enabled,
		AWSAccountResourceID:       configData.AWSAccountResourceID,
		Regions:                    configData.Regions,
		ServiceConfigurationBlocks: configData.ServiceConfigurationBlocks,
	})
}

func (r *resourceAWSCloudWatchScrapeJob) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data resourceAWSCloudWatchScrapeJobModel
	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.State.Set(ctx, nil)
}
