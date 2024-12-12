package fleetmanagement

import (
	"context"

	"connectrpc.com/connect"
	collectorv1 "github.com/grafana/fleet-management-api/api/gen/proto/go/collector/v1"
	"github.com/grafana/fleet-management-api/api/gen/proto/go/collector/v1/collectorv1connect"
	"github.com/grafana/terraform-provider-grafana/v3/internal/common"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const (
	collectorIDField  = "id"
	collectorTypeName = "grafana_fleet_management_collector"
)

var (
	collectorResourceID = common.NewResourceID(common.StringIDField(collectorIDField))
)

var (
	_ resource.Resource                = &collectorResource{}
	_ resource.ResourceWithConfigure   = &collectorResource{}
	_ resource.ResourceWithImportState = &collectorResource{}
)

type collectorResource struct {
	client collectorv1connect.CollectorServiceClient
}

func newCollectorResource() *common.Resource {
	return common.NewResource(
		common.CategoryFleetManagement,
		collectorTypeName,
		collectorResourceID,
		&collectorResource{},
	)
}

func (r *collectorResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil || r.client != nil {
		return
	}

	client, err := withClientForResource(req, resp)
	if err != nil {
		return
	}

	r.client = client.CollectorServiceClient
}

func (r *collectorResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = collectorTypeName
}

func (r *collectorResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "ID of the collector",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"attribute_overrides": schema.MapAttribute{
				Description: "Attribute overrides for the collector",
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Default:     mapdefault.StaticValue(types.MapValueMust(types.StringType, map[string]attr.Value{})),
			},
			"enabled": schema.BoolAttribute{
				Description: "Whether the collector is enabled or not",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
		},
	}
}

func (r *collectorResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	getReq := &collectorv1.GetCollectorRequest{
		Id: req.ID,
	}
	getResp, err := r.client.GetCollector(ctx, connect.NewRequest(getReq))
	if err != nil {
		resp.Diagnostics.AddError("Failed to get collector", err.Error())
		return
	}

	state := collectorMessageToModel(getResp.Msg)
	diags := resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *collectorResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	data := &collectorModel{}
	diags := req.Plan.Get(ctx, data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	collector, err := collectorModelToMessage(data)
	if err != nil {
		resp.Diagnostics.AddError("Invalid input", err.Error())
		return
	}

	createReq := &collectorv1.CreateCollectorRequest{
		Collector: collector,
	}
	_, err = r.client.CreateCollector(ctx, connect.NewRequest(createReq))
	if err != nil {
		resp.Diagnostics.AddError("Failed to create collector", err.Error())
		return
	}

	getReq := &collectorv1.GetCollectorRequest{
		Id: collector.Id,
	}
	getResp, err := r.client.GetCollector(ctx, connect.NewRequest(getReq))
	if err != nil {
		resp.Diagnostics.AddError("Failed to get collector", err.Error())
		return
	}

	state := collectorMessageToModel(getResp.Msg)
	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *collectorResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	data := &collectorModel{}
	diags := req.State.Get(ctx, data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	getReq := &collectorv1.GetCollectorRequest{
		Id: data.ID.ValueString(),
	}
	getResp, err := r.client.GetCollector(ctx, connect.NewRequest(getReq))
	if err != nil {
		resp.Diagnostics.AddError("Failed to get collector", err.Error())
		return
	}

	state := collectorMessageToModel(getResp.Msg)
	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
}

func (r *collectorResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	data := &collectorModel{}
	diags := req.Plan.Get(ctx, data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	collector, err := collectorModelToMessage(data)
	if err != nil {
		resp.Diagnostics.AddError("Invalid input", err.Error())
		return
	}

	updateReq := &collectorv1.UpdateCollectorRequest{
		Collector: collector,
	}
	_, err = r.client.UpdateCollector(ctx, connect.NewRequest(updateReq))
	if err != nil {
		resp.Diagnostics.AddError("Failed to update collector", err.Error())
		return
	}

	getReq := &collectorv1.GetCollectorRequest{
		Id: collector.Id,
	}
	getResp, err := r.client.GetCollector(ctx, connect.NewRequest(getReq))
	if err != nil {
		resp.Diagnostics.AddError("Failed to get collector", err.Error())
		return
	}

	state := collectorMessageToModel(getResp.Msg)
	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *collectorResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	data := &collectorModel{}
	diags := req.State.Get(ctx, data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	deleteReq := &collectorv1.DeleteCollectorRequest{
		Id: data.ID.ValueString(),
	}
	_, err := r.client.DeleteCollector(ctx, connect.NewRequest(deleteReq))
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete collector", err.Error())
		return
	}
}
