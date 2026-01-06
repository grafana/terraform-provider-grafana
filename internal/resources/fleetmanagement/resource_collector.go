package fleetmanagement

import (
	"context"
	"sync"

	"connectrpc.com/connect"
	collectorv1 "github.com/grafana/fleet-management-api/api/gen/proto/go/collector/v1"
	"github.com/grafana/fleet-management-api/api/gen/proto/go/collector/v1/collectorv1connect"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
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

	// Cache for ListCollectors result for plan/refresh
	listOnce       sync.Once
	collectorCache map[string]*collectorv1.Collector
	listErr        error
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
		Description: `
Manages Grafana Fleet Management collectors.

* [Official documentation](https://grafana.com/docs/grafana-cloud/send-data/fleet-management/)
* [API documentation](https://grafana.com/docs/grafana-cloud/send-data/fleet-management/api-reference/collector-api/)
* [Step-by-step guide](https://grafana.com/docs/grafana-cloud/as-code/infrastructure-as-code/terraform/terraform-fleet-management/)

Required access policy scopes:

* fleet-management:read
* fleet-management:write
`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "ID of the collector",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"remote_attributes": schema.MapAttribute{
				Description: "Remote attributes for the collector",
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Default:     mapdefault.StaticValue(types.MapValueMust(types.StringType, map[string]attr.Value{})),
			},
			"enabled": schema.BoolAttribute{
				Description: "Whether remote configuration for the collector is enabled or not. If the collector is disabled, " +
					"it will receive empty configurations from the Fleet Management service",
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(true),
			},
			"collector_type": schema.StringAttribute{
				Description: "Type of the collector. Must be one of: ALLOY, OTEL",
				Required:    true,
				Computed:    true,
				Validators: []validator.String{
					stringvalidator.OneOf("ALLOY", "OTEL"),
				},
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

	// Invalidate cache after import to ensure subsequent reads get fresh data
	r.resetCache()

	state, diags := collectorMessageToResourceModel(ctx, getResp.Msg)
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

func (r *collectorResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	data := &collectorResourceModel{}
	diags := req.Plan.Get(ctx, data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	collector, diags := collectorResourceModelToMessage(ctx, data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := &collectorv1.CreateCollectorRequest{
		Collector: collector,
	}
	_, err := r.client.CreateCollector(ctx, connect.NewRequest(createReq))
	if err != nil {
		resp.Diagnostics.AddError("Failed to create collector", err.Error())
		return
	}

	// Invalidate cache after mutation
	r.resetCache()

	getReq := &collectorv1.GetCollectorRequest{
		Id: collector.Id,
	}
	getResp, err := r.client.GetCollector(ctx, connect.NewRequest(getReq))
	if err != nil {
		resp.Diagnostics.AddError("Failed to get collector", err.Error())
		return
	}

	state, diags := collectorMessageToResourceModel(ctx, getResp.Msg)
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

func (r *collectorResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	data := &collectorResourceModel{}
	diags := req.State.Get(ctx, data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Use cached ListCollectors result for plan/refresh
	r.listOnce.Do(func() {
		listReq := &collectorv1.ListCollectorsRequest{}
		listResp, err := r.client.ListCollectors(ctx, connect.NewRequest(listReq))
		if err != nil {
			r.listErr = err
			return
		}

		r.collectorCache = make(map[string]*collectorv1.Collector, len(listResp.Msg.Collectors))
		for _, collector := range listResp.Msg.Collectors {
			r.collectorCache[collector.Id] = collector
		}
	})

	if r.listErr != nil {
		resp.Diagnostics.AddError("Failed to list collectors", r.listErr.Error())
		return
	}

	collectorID := data.ID.ValueString()
	collector, found := r.collectorCache[collectorID]
	if !found {
		resp.Diagnostics.AddWarning(
			"Collector not found during refresh",
			"Automatically removing resource from Terraform state.",
		)
		resp.State.RemoveResource(ctx)
		return
	}

	state, diags := collectorMessageToResourceModel(ctx, collector)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
}

func (r *collectorResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	data := &collectorResourceModel{}
	diags := req.Plan.Get(ctx, data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	collector, diags := collectorResourceModelToMessage(ctx, data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateReq := &collectorv1.UpdateCollectorRequest{
		Collector: collector,
	}
	_, err := r.client.UpdateCollector(ctx, connect.NewRequest(updateReq))
	if err != nil {
		resp.Diagnostics.AddError("Failed to update collector", err.Error())
		return
	}

	// Invalidate cache after mutation
	r.resetCache()

	getReq := &collectorv1.GetCollectorRequest{
		Id: collector.Id,
	}
	getResp, err := r.client.GetCollector(ctx, connect.NewRequest(getReq))
	if err != nil {
		resp.Diagnostics.AddError("Failed to get collector", err.Error())
		return
	}

	state, diags := collectorMessageToResourceModel(ctx, getResp.Msg)
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

func (r *collectorResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	data := &collectorResourceModel{}
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

	// Invalidate cache after mutation
	r.resetCache()
}

// resetCache invalidates the ListCollectors cache so the next Read will fetch fresh data
func (r *collectorResource) resetCache() {
	r.listOnce = sync.Once{}
	r.collectorCache = nil
	r.listErr = nil
}
