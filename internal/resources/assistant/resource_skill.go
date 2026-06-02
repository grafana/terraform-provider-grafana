package assistant

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common/assistantapi"
	"github.com/grafana/terraform-provider-grafana/v4/internal/util"
)

const resourceSkillName = "grafana_assistant_skill"

var resourceSkillID = common.NewResourceID(common.StringIDField("id"))

type skillResource struct {
	client *assistantapi.Client
}

type skillAllowedToolModel struct {
	IntegrationID types.String `tfsdk:"integration_id"`
	ToolName      types.String `tfsdk:"tool_name"`
}

type skillModel struct {
	ID                     types.String `tfsdk:"id"`
	Scope                  types.String `tfsdk:"scope"`
	Name                   types.String `tfsdk:"name"`
	Body                   types.String `tfsdk:"body"`
	IncludeInKnowledgebase types.Bool   `tfsdk:"include_in_knowledgebase"`
	ContextItems           types.String `tfsdk:"context_items"`
	AllowedTools           types.List   `tfsdk:"allowed_tools"`
}

var skillAllowedToolAttrTypes = map[string]attr.Type{
	"integration_id": types.StringType,
	"tool_name":      types.StringType,
}

func makeResourceSkill() *common.Resource {
	return common.NewResource(
		common.CategoryGrafanaAssistant,
		resourceSkillName,
		resourceSkillID,
		&skillResource{},
	)
}

func (r *skillResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = resourceSkillName
}

func (r *skillResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Grafana Assistant skill.",
		Attributes: map[string]schema.Attribute{
			"id":    idAttribute(),
			"scope": scopeAttribute(),
			"name":  schema.StringAttribute{Description: "The skill name.", Required: true},
			"body":  schema.StringAttribute{Description: "The skill content.", Required: true},
			"include_in_knowledgebase": schema.BoolAttribute{
				Description: "Whether the skill is included in the knowledgebase.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"context_items": schema.StringAttribute{
				Description: "Optional JSON array of context items referenced by the skill.",
				Optional:    true,
			},
		},
		Blocks: map[string]schema.Block{
			"allowed_tools": schema.ListNestedBlock{
				Description: "MCP tools to auto-approve when this skill is invoked.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"integration_id": schema.StringAttribute{Description: "Integration UUID.", Required: true},
						"tool_name":      schema.StringAttribute{Description: "MCP tool name.", Required: true},
					},
				},
			},
		},
	}
}

func (r *skillResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil || r.client != nil {
		return
	}
	client, err := withClientForResource(req, resp)
	if err != nil {
		return
	}
	r.client = client
}

func (r *skillResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan skillModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body, diags := skillPlanToCreate(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	created, err := r.client.CreateSkill(ctx, body)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create assistant skill", err.Error())
		return
	}

	state, stateDiags := skillToModel(ctx, created)
	resp.Diagnostics.Append(stateDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *skillResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state skillModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	skill, err := r.client.GetSkill(ctx, state.ID.ValueString())
	if err != nil {
		if errors.Is(err, assistantapi.ErrNotFound) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read assistant skill", err.Error())
		return
	}

	model, diags := skillToModel(ctx, skill)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
}

func (r *skillResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state skillModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body, diags := skillPlanToUpdate(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	updated, err := r.client.UpdateSkill(ctx, state.ID.ValueString(), state.Scope.ValueString(), body)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update assistant skill", err.Error())
		return
	}

	model, stateDiags := skillToModel(ctx, updated)
	resp.Diagnostics.Append(stateDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
}

func (r *skillResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state skillModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteSkill(ctx, state.ID.ValueString(), state.Scope.ValueString()); err != nil {
		resp.Diagnostics.AddError("Failed to delete assistant skill", err.Error())
	}
}

func (r *skillResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	skill, err := r.client.GetSkill(ctx, req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Failed to import assistant skill", err.Error())
		return
	}
	model, diags := skillToModel(ctx, skill)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
}

func skillPlanToCreate(ctx context.Context, plan skillModel) (assistantapi.SkillCreate, diag.Diagnostics) {
	var diags diag.Diagnostics
	contextItems, ctxDiags := rawJSONFromString(ctx, plan.ContextItems)
	diags.Append(ctxDiags...)
	allowedTools, toolsDiags := allowedToolsFromList(ctx, plan.AllowedTools)
	diags.Append(toolsDiags...)

	include := plan.IncludeInKnowledgebase.ValueBool()
	scope := plan.Scope.ValueString()
	return assistantapi.SkillCreate{
		Name:                   plan.Name.ValueString(),
		Body:                   plan.Body.ValueString(),
		IncludeInKnowledgebase: util.Ptr(include),
		ContextItems:           contextItems,
		Scope:                  &scope,
		AllowedTools:           allowedTools,
	}, diags
}

func skillPlanToUpdate(ctx context.Context, plan skillModel) (assistantapi.SkillUpdate, diag.Diagnostics) {
	var diags diag.Diagnostics
	var contextItems *json.RawMessage
	if !plan.ContextItems.IsNull() && !plan.ContextItems.IsUnknown() {
		raw, ctxDiags := rawJSONFromString(ctx, plan.ContextItems)
		diags.Append(ctxDiags...)
		contextItems = &raw
	}
	var allowedTools *[]assistantapi.AllowedTool
	if !plan.AllowedTools.IsNull() && !plan.AllowedTools.IsUnknown() {
		tools, toolsDiags := allowedToolsFromList(ctx, plan.AllowedTools)
		diags.Append(toolsDiags...)
		allowedTools = &tools
	}

	include := plan.IncludeInKnowledgebase.ValueBool()
	scope := plan.Scope.ValueString()
	return assistantapi.SkillUpdate{
		Name:                   util.Ptr(plan.Name.ValueString()),
		Body:                   util.Ptr(plan.Body.ValueString()),
		IncludeInKnowledgebase: util.Ptr(include),
		ContextItems:           contextItems,
		Scope:                  &scope,
		AllowedTools:           allowedTools,
	}, diags
}

func allowedToolsFromList(ctx context.Context, list types.List) ([]assistantapi.AllowedTool, diag.Diagnostics) {
	if list.IsNull() || list.IsUnknown() {
		return nil, nil
	}
	var tools []skillAllowedToolModel
	var diags diag.Diagnostics
	diags.Append(list.ElementsAs(ctx, &tools, false)...)
	if diags.HasError() {
		return nil, diags
	}
	result := make([]assistantapi.AllowedTool, len(tools))
	for i, t := range tools {
		result[i] = assistantapi.AllowedTool{
			IntegrationID: t.IntegrationID.ValueString(),
			ToolName:      t.ToolName.ValueString(),
		}
	}
	return result, diags
}

func skillToModel(ctx context.Context, skill assistantapi.Skill) (skillModel, diag.Diagnostics) {
	var diags diag.Diagnostics
	allowedTools, toolsDiags := allowedToolsToList(ctx, skill.AllowedTools)
	diags.Append(toolsDiags...)

	return skillModel{
		ID:                     types.StringValue(skill.ID),
		Scope:                  types.StringValue(skill.Scope),
		Name:                   types.StringValue(skill.Name),
		Body:                   types.StringValue(skill.Body),
		IncludeInKnowledgebase: types.BoolValue(skill.IncludeInKnowledgebase),
		ContextItems:           stringFromRawJSON(skill.ContextItems),
		AllowedTools:           allowedTools,
	}, diags
}

func allowedToolsToList(ctx context.Context, tools []assistantapi.AllowedTool) (types.List, diag.Diagnostics) {
	if len(tools) == 0 {
		return types.ListNull(types.ObjectType{AttrTypes: skillAllowedToolAttrTypes}), nil
	}
	values := make([]skillAllowedToolModel, len(tools))
	for i, t := range tools {
		values[i] = skillAllowedToolModel{
			IntegrationID: types.StringValue(t.IntegrationID),
			ToolName:      types.StringValue(t.ToolName),
		}
	}
	return types.ListValueFrom(ctx, types.ObjectType{AttrTypes: skillAllowedToolAttrTypes}, values)
}
