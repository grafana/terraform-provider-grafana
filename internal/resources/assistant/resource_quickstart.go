package assistant

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common/assistantapi"
	"github.com/grafana/terraform-provider-grafana/v4/internal/util"
)

const resourceQuickstartName = "grafana_assistant_quickstart"

var resourceQuickstartID = common.NewResourceID(common.StringIDField("id"))

type quickstartResource struct {
	client *assistantapi.Client
}

type quickstartModel struct {
	ID           types.String `tfsdk:"id"`
	Scope        types.String `tfsdk:"scope"`
	Title        types.String `tfsdk:"title"`
	Prompt       types.String `tfsdk:"prompt"`
	ContextItems types.String `tfsdk:"context_items"`
	Enabled      types.Bool   `tfsdk:"enabled"`
}

func makeResourceQuickstart() *common.Resource {
	return common.NewResource(
		common.CategoryGrafanaAssistant,
		resourceQuickstartName,
		resourceQuickstartID,
		&quickstartResource{},
	)
}

func (r *quickstartResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = resourceQuickstartName
}

func (r *quickstartResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Grafana Assistant quickstart prompt shown to users.",
		Attributes: map[string]schema.Attribute{
			"id":     idAttribute(),
			"scope":  scopeAttribute(),
			"title":  schema.StringAttribute{Description: "Optional title for the quickstart.", Optional: true},
			"prompt": schema.StringAttribute{Description: "The quickstart question text.", Required: true},
			"context_items": schema.StringAttribute{
				Description: "Optional JSON array of context items for the quickstart.",
				Optional:    true,
			},
			"enabled": enabledAttribute(),
		},
	}
}

func (r *quickstartResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil || r.client != nil {
		return
	}
	client, err := withClientForResource(req, resp)
	if err != nil {
		return
	}
	r.client = client
}

func (r *quickstartResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan quickstartModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body, diags := quickstartPlanToCreate(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	created, err := r.client.CreateQuickstart(ctx, body)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create assistant quickstart", err.Error())
		return
	}

	state, stateDiags := quickstartToModel(created)
	resp.Diagnostics.Append(stateDiags...)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *quickstartResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state quickstartModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	quickstart, err := r.client.GetQuickstart(ctx, state.ID.ValueString())
	if err != nil {
		if errors.Is(err, assistantapi.ErrNotFound) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read assistant quickstart", err.Error())
		return
	}

	model, diags := quickstartToModel(quickstart)
	resp.Diagnostics.Append(diags...)
	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
}

func (r *quickstartResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state quickstartModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body, diags := quickstartPlanToUpdate(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	updated, err := r.client.UpdateQuickstart(ctx, state.ID.ValueString(), state.Scope.ValueString(), body)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update assistant quickstart", err.Error())
		return
	}

	model, stateDiags := quickstartToModel(updated)
	resp.Diagnostics.Append(stateDiags...)
	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
}

func (r *quickstartResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state quickstartModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteQuickstart(ctx, state.ID.ValueString(), state.Scope.ValueString()); err != nil {
		resp.Diagnostics.AddError("Failed to delete assistant quickstart", err.Error())
	}
}

func (r *quickstartResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	quickstart, err := r.client.GetQuickstart(ctx, req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Failed to import assistant quickstart", err.Error())
		return
	}
	model, diags := quickstartToModel(quickstart)
	resp.Diagnostics.Append(diags...)
	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
}

func quickstartPlanToCreate(ctx context.Context, plan quickstartModel) (assistantapi.QuickstartCreate, diag.Diagnostics) {
	var diags diag.Diagnostics
	contextItems, ctxDiags := rawJSONFromString(ctx, plan.ContextItems)
	diags.Append(ctxDiags...)

	var title *string
	if !plan.Title.IsNull() && !plan.Title.IsUnknown() {
		title = util.Ptr(plan.Title.ValueString())
	}
	enabled := plan.Enabled.ValueBool()
	return assistantapi.QuickstartCreate{
		Scope:        plan.Scope.ValueString(),
		Title:        title,
		Prompt:       plan.Prompt.ValueString(),
		ContextItems: contextItems,
		Enabled:      util.Ptr(enabled),
	}, diags
}

func quickstartPlanToUpdate(ctx context.Context, plan quickstartModel) (assistantapi.QuickstartUpdate, diag.Diagnostics) {
	var diags diag.Diagnostics
	var contextItems *json.RawMessage
	if !plan.ContextItems.IsNull() && !plan.ContextItems.IsUnknown() {
		raw, ctxDiags := rawJSONFromString(ctx, plan.ContextItems)
		diags.Append(ctxDiags...)
		contextItems = &raw
	}
	var title *string
	if !plan.Title.IsNull() && !plan.Title.IsUnknown() {
		title = util.Ptr(plan.Title.ValueString())
	}
	enabled := plan.Enabled.ValueBool()
	return assistantapi.QuickstartUpdate{
		Scope:        plan.Scope.ValueString(),
		Title:        title,
		Prompt:       util.Ptr(plan.Prompt.ValueString()),
		ContextItems: contextItems,
		Enabled:      util.Ptr(enabled),
	}, diags
}

func quickstartToModel(q assistantapi.Quickstart) (quickstartModel, diag.Diagnostics) {
	enabled := true
	if q.Enabled != nil {
		enabled = *q.Enabled
	}
	title := types.StringNull()
	if q.Title != nil {
		title = types.StringValue(*q.Title)
	}
	return quickstartModel{
		ID:           types.StringValue(q.ID),
		Scope:        types.StringValue(q.Scope),
		Title:        title,
		Prompt:       types.StringValue(q.Prompt),
		ContextItems: stringFromRawJSON(q.ContextItems),
		Enabled:      types.BoolValue(enabled),
	}, nil
}
