package cloudprovider

import (
	"context"
	"fmt"
	"strings"

	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common/cloudproviderapi"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

var (
	resourceAWSResourceMetadataScrapeJobTerraformName = "grafana_cloud_provider_aws_resource_metadata_scrape_job"
	resourceAWSResourceMetadataScrapeJobTerraformID   = common.NewResourceID(common.StringIDField("stack_id"), common.StringIDField("name"))
)

type resourceAWSResourceMetadataScrapeJob struct {
	client *cloudproviderapi.Client
}

func makeResourceAWSResourceMetadataScrapeJob() *common.Resource {
	return common.NewResource(
		common.CategoryCloudProvider,
		"grafana_cloud_provider_aws_resource_metadata_scrape_job",
		resourceAWSResourceMetadataScrapeJobTerraformID,
		&resourceAWSResourceMetadataScrapeJob{},
	)
}

func (r *resourceAWSResourceMetadataScrapeJob) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r resourceAWSResourceMetadataScrapeJob) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = resourceAWSResourceMetadataScrapeJobTerraformName
}

func (r resourceAWSResourceMetadataScrapeJob) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `
This resource allows you to scrape AWS resource metadata such as ARN and tags as info metrics in Grafana Cloud without needing to run your own infrastructure.
Use this resource if you aren't using ` + "`grafana_cloud_provider_aws_cloudwatch_scrape_job`" + `, but still want to have AWS resource metadata available 
in Grafana Cloud, for example for use with our AWS Metrics Streams integration and/or Knowledge Graph features.

See the [Grafana Provider configuration docs](https://registry.terraform.io/providers/grafana/grafana/latest/docs#managing-cloud-provider)
for information on authentication and required access policy scopes.

* [Official Grafana Cloud documentation](https://grafana.com/docs/grafana-cloud/monitor-infrastructure/monitor-cloud-provider/aws/)
`,
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
				Description: "The name of the AWS Resource Metadata Scrape Job. Part of the Terraform Resource ID.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"enabled": schema.BoolAttribute{
				Description: "Whether the AWS Resource Metadata Scrape Job is enabled or not. Defaults to `true`.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"aws_account_resource_id": schema.StringAttribute{
				Description: "The ID assigned by the Grafana Cloud Provider API to an AWS Account resource that should be associated with this Resource Metadata Scrape Job. This can be provided by the `resource_id` attribute of the `grafana_cloud_provider_aws_account` resource.",
				Required:    true,
			},
			"regions_subset_override": schema.SetAttribute{
				Description: "A subset of the regions that are configured in the associated AWS Account resource to apply to this scrape job. If not set or empty, all of the Account resource's regions are scraped.",
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Default:     setdefault.StaticValue(types.SetValueMust(types.StringType, []attr.Value{})),
			},
			"disabled_reason": schema.StringAttribute{
				Description: "When the AWS Resource Metadata Scrape Job is disabled, this will show the reason that it is in that state.",
				Computed:    true,
			},
			"static_labels": schema.MapAttribute{
				Description: "A set of static labels to add to all metrics exported by this scrape job.",
				Optional:    true,
				Computed:    true,
				ElementType: basetypes.StringType{},
				Default:     mapdefault.StaticValue(types.MapValueMust(types.StringType, map[string]attr.Value{})),
			},
		},
		Blocks: map[string]schema.Block{
			"service": schema.ListNestedBlock{
				Description: "One or more configuration blocks to configure AWS services for the Resource Metadata Scrape Job to scrape. Each block must have a distinct `name` attribute. When accessing this as an attribute reference, it is a list of objects.",
				Validators: []validator.List{
					listvalidator.SizeAtLeast(1),
					awsResourceMetadataScrapeJobNoDuplicateServiceNamesValidator{},
				},
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Description: "The name of the service to scrape. See https://grafana.com/docs/grafana-cloud/monitor-infrastructure/monitor-cloud-provider/aws/cloudwatch-metrics/services/ for supported services.",
							Required:    true,
						},
						"scrape_interval_seconds": schema.Int64Attribute{
							Description: "The interval in seconds to scrape the service. See https://grafana.com/docs/grafana-cloud/monitor-infrastructure/monitor-cloud-provider/aws/cloudwatch-metrics/services/ for supported scrape intervals. Defaults to `300`.",
							Optional:    true,
							Computed:    true,
							Default:     int64default.StaticInt64(300),
						},
					},
					Blocks: map[string]schema.Block{
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
		},
	}
}

func (r resourceAWSResourceMetadataScrapeJob) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, ":", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Invalid ID: %s", req.ID))
		return
	}
	stackID := parts[0]
	jobName := parts[1]

	jobResp, err := r.client.GetAWSResourceMetadataScrapeJob(
		ctx,
		stackID,
		jobName,
	)
	if err != nil {
		resp.Diagnostics.AddError("Failed to get AWS Resource Metadata scrape job", err.Error())
		return
	}

	jobTF, diags := generateAWSResourceMetadataScrapeJobTFResourceModel(ctx, stackID, jobResp)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.State.Set(ctx, jobTF)
}

func (r resourceAWSResourceMetadataScrapeJob) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data awsResourceMetadataScrapeJobTFResourceModel
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

	jobResp, err := r.client.CreateAWSResourceMetadataScrapeJob(ctx, data.StackID.ValueString(), jobReq)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create AWS Resource Metadata scrape job", err.Error())
		return
	}

	jobTF, diags := generateAWSResourceMetadataScrapeJobTFResourceModel(ctx, data.StackID.ValueString(), jobResp)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.State.Set(ctx, &jobTF)
}

func (r resourceAWSResourceMetadataScrapeJob) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data awsResourceMetadataScrapeJobTFResourceModel
	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	jobResp, err := r.client.GetAWSResourceMetadataScrapeJob(
		ctx,
		data.StackID.ValueString(),
		data.Name.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError("Failed to get AWS Resource Metadata scrape job", err.Error())
		return
	}

	jobTF, diags := generateAWSResourceMetadataScrapeJobTFResourceModel(ctx, data.StackID.ValueString(), jobResp)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.State.Set(ctx, jobTF)
}

func (r resourceAWSResourceMetadataScrapeJob) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	// This must be a pointer because ModifyPlan is called even on resource creation, when no state exists yet.
	var stateData *awsResourceMetadataScrapeJobTFResourceModel
	diags := req.State.Get(ctx, &stateData)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var planData *awsResourceMetadataScrapeJobTFResourceModel
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

func (r resourceAWSResourceMetadataScrapeJob) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var planData awsResourceMetadataScrapeJobTFResourceModel
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

	jobResp, err := r.client.UpdateAWSResourceMetadataScrapeJob(ctx, planData.StackID.ValueString(), planData.Name.ValueString(), jobReq)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update AWS Resource Metadata scrape job", err.Error())
		return
	}

	jobTF, diags := generateAWSResourceMetadataScrapeJobTFResourceModel(ctx, planData.StackID.ValueString(), jobResp)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.State.Set(ctx, jobTF)
}

func (r resourceAWSResourceMetadataScrapeJob) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data awsResourceMetadataScrapeJobTFResourceModel
	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteAWSResourceMetadataScrapeJob(
		ctx,
		data.StackID.ValueString(),
		data.Name.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete AWS Resource Metadata scrape job", err.Error())
		return
	}

	resp.State.Set(ctx, nil)
}
