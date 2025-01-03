package cloudprovider

import (
	"context"
	"fmt"
	"strings"

	"github.com/grafana/terraform-provider-grafana/v3/internal/common"
	"github.com/grafana/terraform-provider-grafana/v3/internal/common/cloudproviderapi"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

var (
	resourceAWSCloudWatchScrapeJobTerraformName = "grafana_cloud_provider_aws_cloudwatch_scrape_job"
	resourceAWSCloudWatchScrapeJobTerraformID   = common.NewResourceID(common.StringIDField("stack_id"), common.StringIDField("name"))
)

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

func (r resourceAWSCloudWatchScrapeJob) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = resourceAWSCloudWatchScrapeJobTerraformName
}

func (r resourceAWSCloudWatchScrapeJob) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The Terraform Resource ID. This has the format \"{{ stack_id }}:{{ name }}\".",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					// See https://developer.hashicorp.com/terraform/plugin/framework/resources/plan-modification#usestateforunknown
					// for details on how this works.
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"stack_id": schema.StringAttribute{
				Description: "The Stack ID of the Grafana Cloud instance. Part of the Terraform Resource ID.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the CloudWatch Scrape Job. Part of the Terraform Resource ID.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"enabled": schema.BoolAttribute{
				Description: "Whether the CloudWatch Scrape Job is enabled or not.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"aws_account_resource_id": schema.StringAttribute{
				Description: "The ID assigned by the Grafana Cloud Provider API to an AWS Account resource that should be associated with this CloudWatch Scrape Job. This can be provided by the `resource_id` attribute of the `grafana_cloud_provider_aws_account` resource.",
				Required:    true,
			},
			"regions_subset_override": schema.SetAttribute{
				Description: "A subset of the regions that are configured in the associated AWS Account resource to apply to this scrape job. If not set or empty, all of the Account resource's regions are scraped.",
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Default:     setdefault.StaticValue(types.SetValueMust(types.StringType, []attr.Value{})),
			},
			"export_tags": schema.BoolAttribute{
				Description: "When enabled, AWS resource tags are exported as Prometheus labels to metrics formatted as `aws_<service_name>_info`.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"disabled_reason": schema.StringAttribute{
				Description: "When the CloudWatch Scrape Job is disabled, this will show the reason that it is in that state.",
				Computed:    true,
			},
		},
		Blocks: map[string]schema.Block{
			"service": schema.ListNestedBlock{
				Description: "One or more configuration blocks to configure AWS services for the CloudWatch Scrape Job to scrape. Each block must have a distinct `name` attribute. When accessing this as an attribute reference, it is a list of objects.",
				Validators: []validator.List{
					listvalidator.SizeAtLeast(1),
					awsCWScrapeJobNoDuplicateServiceNamesValidator{},
				},
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Description: "The name of the service to scrape. See https://grafana.com/docs/grafana-cloud/monitor-infrastructure/aws/cloudwatch-metrics/services/ for supported services.",
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
							Computed:    true,
							ElementType: types.StringType,
							Default:     setdefault.StaticValue(types.SetValueMust(types.StringType, []attr.Value{})),
						},
					},
					Blocks: map[string]schema.Block{
						"metric": schema.ListNestedBlock{
							Description: "One or more configuration blocks to configure metrics and their statistics to scrape. Please note that AWS metric names must be supplied, and not their PromQL counterparts. Each block must represent a distinct metric name. When accessing this as an attribute reference, it is a list of objects.",
							Validators: []validator.List{
								listvalidator.SizeAtLeast(1),
								awsCWScrapeJobNoDuplicateMetricNamesValidator{},
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
						"resource_discovery_tag_filter": schema.ListNestedBlock{
							Description: "One or more configuration blocks to configure tag filters applied to discovery of resource entities in the associated AWS account. When accessing this as an attribute reference, it is a list of objects.",
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
			"custom_namespace": schema.ListNestedBlock{
				Description: "Zero or more configuration blocks to configure custom namespaces for the CloudWatch Scrape Job to scrape. Each block must have a distinct `name` attribute. When accessing this as an attribute reference, it is a list of objects.",
				Validators: []validator.List{
					awsCWScrapeJobNoDuplicateCustomNamespaceNamesValidator{},
				},
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Description: "The name of the custom namespace to scrape.",
							Required:    true,
						},
						"scrape_interval_seconds": schema.Int64Attribute{
							Description: "The interval in seconds to scrape the custom namespace.",
							Optional:    true,
							Computed:    true,
							Default:     int64default.StaticInt64(300),
						},
					},
					Blocks: map[string]schema.Block{
						"metric": schema.ListNestedBlock{
							Description: "One or more configuration blocks to configure metrics and their statistics to scrape. Each block must represent a distinct metric name. When accessing this as an attribute reference, it is a list of objects.",
							Validators: []validator.List{
								listvalidator.SizeAtLeast(1),
								awsCWScrapeJobNoDuplicateMetricNamesValidator{},
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
					},
				},
			},
			"static_label": schema.ListNestedBlock{
				Description: "Zero or more configuration blocks to configure static labels to add to all metrics exported by this scrape job.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"label": schema.StringAttribute{
							Description: "The label.",
							Required:    true,
						},
						"value": schema.StringAttribute{
							Description: "The value of the label.",
							Required:    true,
						},
					},
				},
			},
		},
	}
}

func (r resourceAWSCloudWatchScrapeJob) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, ":", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Invalid ID: %s", req.ID))
		return
	}
	stackID := parts[0]
	jobName := parts[1]

	jobResp, err := r.client.GetAWSCloudWatchScrapeJob(
		ctx,
		stackID,
		jobName,
	)
	if err != nil {
		resp.Diagnostics.AddError("Failed to get AWS CloudWatch scrape job", err.Error())
		return
	}

	jobTF, diags := generateCloudWatchScrapeJobTFResourceModel(ctx, stackID, jobResp)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.State.Set(ctx, jobTF)
}

func (r resourceAWSCloudWatchScrapeJob) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data awsCWScrapeJobTFResourceModel
	diags := req.Plan.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	jobReq, diags := data.toClientModel(ctx)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	jobResp, err := r.client.CreateAWSCloudWatchScrapeJob(ctx, data.StackID.ValueString(), jobReq)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create AWS CloudWatch scrape job", err.Error())
		return
	}

	jobTF, diags := generateCloudWatchScrapeJobTFResourceModel(ctx, data.StackID.ValueString(), jobResp)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.State.Set(ctx, &jobTF)
}

func (r resourceAWSCloudWatchScrapeJob) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data awsCWScrapeJobTFResourceModel
	diags := req.State.Get(ctx, &data)
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

	jobTF, diags := generateCloudWatchScrapeJobTFResourceModel(ctx, data.StackID.ValueString(), jobResp)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.State.Set(ctx, jobTF)
}

func (r resourceAWSCloudWatchScrapeJob) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	// This must be a pointer because ModifyPlan is called even on resource creation, when no state exists yet.
	var stateData *awsCWScrapeJobTFResourceModel
	diags := req.State.Get(ctx, &stateData)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var planData *awsCWScrapeJobTFResourceModel
	diags = req.Plan.Get(ctx, &planData)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// resource is being deleted
	if planData == nil {
		return
	}

	// This helps reduce the occurrences of Unknown states for the disabled_reason attribute
	// by tying it to how the enabled attribute will change.
	switch {
	case (stateData == nil || stateData.Enabled.ValueBool()) && !planData.Enabled.ValueBool():
		resp.Plan.SetAttribute(ctx, path.Root("disabled_reason"), basetypes.NewStringUnknown())
	case (stateData == nil || !stateData.Enabled.ValueBool()) && planData.Enabled.ValueBool():
		resp.Plan.SetAttribute(ctx, path.Root("disabled_reason"), basetypes.NewStringValue(""))
	default:
		resp.Plan.SetAttribute(ctx, path.Root("disabled_reason"), basetypes.NewStringValue(stateData.DisabledReason.ValueString()))
	}
}

func (r resourceAWSCloudWatchScrapeJob) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var planData awsCWScrapeJobTFResourceModel
	diags := req.Plan.Get(ctx, &planData)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	jobReq, diags := planData.toClientModel(ctx)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	jobResp, err := r.client.UpdateAWSCloudWatchScrapeJob(ctx, planData.StackID.ValueString(), planData.Name.ValueString(), jobReq)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update AWS CloudWatch scrape job", err.Error())
		return
	}

	jobTF, diags := generateCloudWatchScrapeJobTFResourceModel(ctx, planData.StackID.ValueString(), jobResp)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.State.Set(ctx, jobTF)
}

func (r resourceAWSCloudWatchScrapeJob) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data awsCWScrapeJobTFResourceModel
	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteAWSCloudWatchScrapeJob(
		ctx,
		data.StackID.ValueString(),
		data.Name.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete AWS CloudWatch scrape job", err.Error())
		return
	}

	resp.State.Set(ctx, nil)
}
