package agento11y

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common/agento11yapi"
)

const resourceEvaluatorName = "grafana_agento11y_evaluator"

var resourceEvaluatorID = common.NewResourceID(common.StringIDField("evaluator_id"))

type evaluatorResource struct {
	client *agento11yapi.Client
}

type evaluatorModel struct {
	ID          types.String         `tfsdk:"id"`
	EvaluatorID types.String         `tfsdk:"evaluator_id"`
	Version     types.String         `tfsdk:"version"`
	Kind        types.String         `tfsdk:"kind"`
	Description types.String         `tfsdk:"description"`
	Config      jsontypes.Normalized `tfsdk:"config"`
	OutputKeys  jsontypes.Normalized `tfsdk:"output_keys"`
}

func makeResourceEvaluator() *common.Resource {
	return common.NewResource(
		common.CategoryAgentObservability,
		resourceEvaluatorName,
		resourceEvaluatorID,
		&evaluatorResource{},
	)
}

func (r *evaluatorResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = resourceEvaluatorName
}

func (r *evaluatorResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Grafana Agent Observability evaluator definition. Evaluators score agent generations or conversations (LLM judge, JSON schema, regex, or heuristic).\n\nRequires a Grafana instance with the `grafana-agento11y-app` plugin installed. " + writePermissionDescription,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The evaluator identifier (equal to `evaluator_id`).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"evaluator_id": schema.StringAttribute{
				Description: "Tenant-unique identifier of the evaluator. Changing this forces a new resource.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"version": schema.StringAttribute{
				Description: "Version label of the evaluator definition.",
				Required:    true,
			},
			"kind": schema.StringAttribute{
				Description: "The evaluator kind. One of `llm_judge`, `json_schema`, `regex`, `heuristic`.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.OneOf("llm_judge", "json_schema", "regex", "heuristic"),
				},
			},
			"description": schema.StringAttribute{
				Description: "Optional human-readable description of the evaluator.",
				Optional:    true,
			},
			"config": schema.StringAttribute{
				CustomType:  jsontypes.NormalizedType{},
				Description: "Kind-specific evaluator configuration, encoded as a JSON object string. The server normalizes this payload, so it is managed from configuration and not refreshed from the API.",
				Required:    true,
			},
			"output_keys": schema.StringAttribute{
				CustomType:  jsontypes.NormalizedType{},
				Description: "JSON array of output key definitions produced by the evaluator (for example `[{\"key\":\"score\",\"type\":\"number\",\"pass_threshold\":0.5}]`). Managed from configuration and not refreshed from the API.",
				Required:    true,
			},
		},
	}
}

func (r *evaluatorResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil || r.client != nil {
		return
	}
	client, err := withClientForResource(req, resp)
	if err != nil {
		return
	}
	r.client = client
}

func (r *evaluatorResource) write(ctx context.Context, plan evaluatorModel) (evaluatorModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	var outputKeys []agento11yapi.OutputKey
	if raw := rawJSONFromNormalized(plan.OutputKeys); len(raw) > 0 {
		if err := json.Unmarshal(raw, &outputKeys); err != nil {
			diags.AddError("Invalid output_keys", "output_keys must be a JSON array of output key objects: "+err.Error())
			return evaluatorModel{}, diags
		}
	}

	created, err := r.client.CreateEvaluator(ctx, agento11yapi.EvaluatorWrite{
		EvaluatorID: plan.EvaluatorID.ValueString(),
		Version:     plan.Version.ValueString(),
		Kind:        plan.Kind.ValueString(),
		Description: plan.Description.ValueString(),
		Config:      rawJSONFromNormalized(plan.Config),
		OutputKeys:  outputKeys,
	})
	if err != nil {
		diags.AddError("Failed to write agent observability evaluator", err.Error())
		return evaluatorModel{}, diags
	}

	// config and output_keys are preserved from the plan because the server
	// normalizes them; refreshing would produce spurious diffs.
	return evaluatorModel{
		ID:          types.StringValue(created.EvaluatorID),
		EvaluatorID: types.StringValue(created.EvaluatorID),
		Version:     types.StringValue(created.Version),
		Kind:        types.StringValue(created.Kind),
		Description: stringValueOrNull(created.Description),
		Config:      plan.Config,
		OutputKeys:  plan.OutputKeys,
	}, diags
}

func (r *evaluatorResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan evaluatorModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	state, diags := r.write(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *evaluatorResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state evaluatorModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	evaluator, err := r.client.GetEvaluator(ctx, state.EvaluatorID.ValueString())
	if err != nil {
		if errors.Is(err, agento11yapi.ErrNotFound) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read agent observability evaluator", err.Error())
		return
	}

	state.ID = types.StringValue(evaluator.EvaluatorID)
	state.EvaluatorID = types.StringValue(evaluator.EvaluatorID)
	state.Version = types.StringValue(evaluator.Version)
	state.Kind = types.StringValue(evaluator.Kind)
	state.Description = stringValueOrNull(evaluator.Description)
	// config and output_keys are intentionally preserved from state.
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *evaluatorResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan evaluatorModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	state, diags := r.write(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *evaluatorResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state evaluatorModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteEvaluator(ctx, state.EvaluatorID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Failed to delete agent observability evaluator", err.Error())
		return
	}
}

func (r *evaluatorResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	evaluator, err := r.client.GetEvaluator(ctx, req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Failed to import agent observability evaluator", err.Error())
		return
	}

	outputKeys := jsontypes.NewNormalizedNull()
	if len(evaluator.OutputKeys) > 0 {
		raw, err := json.Marshal(evaluator.OutputKeys)
		if err != nil {
			resp.Diagnostics.AddError("Failed to import agent observability evaluator", err.Error())
			return
		}
		outputKeys = jsontypes.NewNormalizedValue(string(raw))
	}

	state := evaluatorModel{
		ID:          types.StringValue(evaluator.EvaluatorID),
		EvaluatorID: types.StringValue(evaluator.EvaluatorID),
		Version:     types.StringValue(evaluator.Version),
		Kind:        types.StringValue(evaluator.Kind),
		Description: stringValueOrNull(evaluator.Description),
		Config:      normalizedFromRawJSON(evaluator.Config),
		OutputKeys:  outputKeys,
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}
