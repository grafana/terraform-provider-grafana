package fleetmanagement

import (
	"context"

	"connectrpc.com/connect"
	pipelinev1 "github.com/grafana/fleet-management-api/api/gen/proto/go/pipeline/v1"
	"github.com/grafana/fleet-management-api/api/gen/proto/go/pipeline/v1/pipelinev1connect"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

const (
	pipelineIDField  = "name"
	pipelineTypeName = "grafana_fleet_management_pipeline"
)

var (
	pipelineResourceID = common.NewResourceID(common.StringIDField(pipelineIDField))
)

var (
	_ resource.Resource                = &pipelineResource{}
	_ resource.ResourceWithConfigure   = &pipelineResource{}
	_ resource.ResourceWithImportState = &pipelineResource{}
)

type pipelineResource struct {
	client pipelinev1connect.PipelineServiceClient
}

func newPipelineResource() *common.Resource {
	return common.NewResource(
		common.CategoryFleetManagement,
		pipelineTypeName,
		pipelineResourceID,
		&pipelineResource{},
	)
}

func (r *pipelineResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil || r.client != nil {
		return
	}

	client, err := withClientForResource(req, resp)
	if err != nil {
		return
	}

	r.client = client.PipelineServiceClient
}

func (r *pipelineResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = pipelineTypeName
}

func (r *pipelineResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `
Manages Grafana Fleet Management pipelines.

* [Official documentation](https://grafana.com/docs/grafana-cloud/send-data/fleet-management/)
* [API documentation](https://grafana.com/docs/grafana-cloud/send-data/fleet-management/api-reference/pipeline-api/)
* [Step-by-step guide](https://grafana.com/docs/grafana-cloud/as-code/infrastructure-as-code/terraform/terraform-fleet-management/)

Required access policy scopes:

* fleet-management:read
* fleet-management:write
`,
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Description: "Name of the pipeline which is the unique identifier for the pipeline",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"contents": schema.StringAttribute{
				CustomType:  PipelineConfigType,
				Description: "Configuration contents of the pipeline to be used by collectors (can be Alloy River syntax or OTel YAML)",
				Required:    true,
			},
			"matchers": schema.ListAttribute{
				CustomType:  ListOfPrometheusMatcherType,
				Description: "Used to match against collectors and assign pipelines to them; follows the syntax of Prometheus Alertmanager matchers",
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Default: listdefault.StaticValue(
					basetypes.NewListValueMust(
						types.StringType,
						[]attr.Value{},
					),
				),
			},
			"enabled": schema.BoolAttribute{
				Description: "Whether the pipeline is enabled for collectors",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"id": schema.StringAttribute{
				Description: "Server-assigned ID of the pipeline",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"config_type": schema.StringAttribute{
				Description: "Type of the config. Must be one of: ALLOY, OTEL. Defaults to ALLOY if not specified.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("ALLOY"),
				Validators: []validator.String{
					stringvalidator.OneOf("ALLOY", "OTEL"),
				},
			},
		},
	}
}

func (r *pipelineResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	getIDReq := &pipelinev1.GetPipelineIDRequest{
		Name: req.ID,
	}
	getIDResp, err := r.client.GetPipelineID(ctx, connect.NewRequest(getIDReq))
	if err != nil {
		resp.Diagnostics.AddError("Failed to get pipeline ID", err.Error())
		return
	}

	getReq := &pipelinev1.GetPipelineRequest{
		Id: getIDResp.Msg.Id,
	}
	getResp, err := r.client.GetPipeline(ctx, connect.NewRequest(getReq))
	if err != nil {
		resp.Diagnostics.AddError("Failed to get pipeline", err.Error())
		return
	}

	state, diags := pipelineMessageToModel(ctx, getResp.Msg)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *pipelineResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	data := &pipelineModel{}
	diags := req.Plan.Get(ctx, data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	pipeline, diags := pipelineModelToMessage(ctx, data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := &pipelinev1.CreatePipelineRequest{
		Pipeline: pipeline,
	}
	createResp, err := r.client.CreatePipeline(ctx, connect.NewRequest(createReq))
	if err != nil {
		resp.Diagnostics.AddError("Failed to create pipeline", err.Error())
		return
	}

	getReq := &pipelinev1.GetPipelineRequest{
		Id: *createResp.Msg.Id,
	}
	getResp, err := r.client.GetPipeline(ctx, connect.NewRequest(getReq))
	if err != nil {
		resp.Diagnostics.AddError("Failed to get pipeline", err.Error())
		return
	}

	state, diags := pipelineMessageToModel(ctx, getResp.Msg)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *pipelineResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	data := &pipelineModel{}
	diags := req.State.Get(ctx, data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	getReq := &pipelinev1.GetPipelineRequest{
		Id: data.ID.ValueString(),
	}
	getResp, err := r.client.GetPipeline(ctx, connect.NewRequest(getReq))
	if connect.CodeOf(err) == connect.CodeNotFound {
		resp.Diagnostics.AddWarning(
			"Pipeline not found during refresh",
			"Automatically removing resource from Terraform state. Original error: "+err.Error(),
		)
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Failed to get pipeline", err.Error())
		return
	}

	state, diags := pipelineMessageToModel(ctx, getResp.Msg)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
}

func (r *pipelineResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	data := &pipelineModel{}
	diags := req.Plan.Get(ctx, data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	pipeline, diags := pipelineModelToMessage(ctx, data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateReq := &pipelinev1.UpdatePipelineRequest{
		Pipeline: pipeline,
	}
	_, err := r.client.UpdatePipeline(ctx, connect.NewRequest(updateReq))
	if err != nil {
		resp.Diagnostics.AddError("Failed to update pipeline", err.Error())
		return
	}

	getReq := &pipelinev1.GetPipelineRequest{
		Id: *pipeline.Id,
	}
	getResp, err := r.client.GetPipeline(ctx, connect.NewRequest(getReq))
	if err != nil {
		resp.Diagnostics.AddError("Failed to get pipeline", err.Error())
		return
	}

	state, diags := pipelineMessageToModel(ctx, getResp.Msg)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *pipelineResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	data := &pipelineModel{}
	diags := req.State.Get(ctx, data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	deleteReq := &pipelinev1.DeletePipelineRequest{
		Id: data.ID.ValueString(),
	}
	_, err := r.client.DeletePipeline(ctx, connect.NewRequest(deleteReq))
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete pipeline", err.Error())
		return
	}
}
