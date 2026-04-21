package asserts

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"time"

	assertsapi "github.com/grafana/grafana-asserts-public-clients/go/gcom"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &customModelRulesResource{}
	_ resource.ResourceWithConfigure   = &customModelRulesResource{}
	_ resource.ResourceWithImportState = &customModelRulesResource{}
)

type customModelRulesModel struct {
	ID    types.String `tfsdk:"id"`
	Name  types.String `tfsdk:"name"`
	Rules []rulesModel `tfsdk:"rules"`
}

type rulesModel struct {
	Entity []entityModel `tfsdk:"entity"`
}

type entityModel struct {
	Type       types.String     `tfsdk:"type"`
	Name       types.String     `tfsdk:"name"`
	Scope      types.Map        `tfsdk:"scope"`
	Lookup     types.Map        `tfsdk:"lookup"`
	EnrichedBy types.List       `tfsdk:"enriched_by"`
	Disabled   types.Bool       `tfsdk:"disabled"`
	DefinedBy  []definedByModel `tfsdk:"defined_by"`
}

type definedByModel struct {
	Query       types.String `tfsdk:"query"`
	Disabled    types.Bool   `tfsdk:"disabled"`
	LabelValues types.Map    `tfsdk:"label_values"`
	Literals    types.Map    `tfsdk:"literals"`
	MetricValue types.String `tfsdk:"metric_value"`
}

type customModelRulesResource struct {
	client  *assertsapi.APIClient
	stackID int64
}

func makeResourceCustomModelRules() *common.Resource {
	return common.NewResource(
		common.CategoryAsserts,
		"grafana_asserts_custom_model_rules",
		common.NewResourceID(common.StringIDField("name")),
		&customModelRulesResource{},
	).WithLister(assertsListerFunction(listCustomModelRules))
}

func (r *customModelRulesResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "grafana_asserts_custom_model_rules"
}

func (r *customModelRulesResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil || r.client != nil {
		return
	}
	client, ok := req.ProviderData.(*common.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Configure Type",
			fmt.Sprintf("Expected *common.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}
	if client.AssertsAPIClient == nil {
		resp.Diagnostics.AddError(
			"Asserts API client not configured",
			"The Grafana provider is missing a configuration for the Asserts API. Ensure stack_id is set in the provider configuration.",
		)
		return
	}
	r.client = client.AssertsAPIClient
	r.stackID = client.GrafanaStackID
}

func (r *customModelRulesResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages Knowledge Graph Custom Model Rules through the Grafana API.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the custom model rules.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
		Blocks: map[string]schema.Block{
			"rules": schema.ListNestedBlock{
				Description: "The rules configuration for the custom model rules.",
				Validators: []validator.List{
					listvalidator.SizeAtLeast(1),
					listvalidator.SizeAtMost(1),
				},
				NestedObject: schema.NestedBlockObject{
					Blocks: map[string]schema.Block{
						"entity": schema.ListNestedBlock{
							Description: "List of entities to define in the custom model rules.",
							Validators: []validator.List{
								listvalidator.SizeAtLeast(1),
							},
							NestedObject: schema.NestedBlockObject{
								Attributes: map[string]schema.Attribute{
									"type": schema.StringAttribute{
										Required:    true,
										Description: "The type of the entity (e.g., Service, Pod, Namespace).",
									},
									"name": schema.StringAttribute{
										Required:    true,
										Description: "The name of the entity.",
									},
									"scope": schema.MapAttribute{
										Optional:    true,
										Description: "Scope labels for the entity.",
										ElementType: types.StringType,
									},
									"lookup": schema.MapAttribute{
										Optional:    true,
										Description: "Lookup mappings for the entity.",
										ElementType: types.StringType,
									},
									"enriched_by": schema.ListAttribute{
										Optional:    true,
										Description: "List of enrichment sources for the entity.",
										ElementType: types.StringType,
									},
									"disabled": schema.BoolAttribute{
										Optional:    true,
										Description: "Whether this entity is disabled.",
									},
								},
								Blocks: map[string]schema.Block{
									"defined_by": schema.ListNestedBlock{
										Description: "List of queries that define this entity.",
										Validators: []validator.List{
											listvalidator.SizeAtLeast(1),
										},
										NestedObject: schema.NestedBlockObject{
											Attributes: map[string]schema.Attribute{
												"query": schema.StringAttribute{
													Required:    true,
													Description: "The Prometheus query that defines this entity.",
												},
												"disabled": schema.BoolAttribute{
													Optional:    true,
													Description: "Whether this rule is disabled. When true, only the 'query' field is used to match an existing rule to disable; other fields are ignored.",
												},
												"label_values": schema.MapAttribute{
													Optional:    true,
													Description: "Label value mappings for the query.",
													ElementType: types.StringType,
												},
												"literals": schema.MapAttribute{
													Optional:    true,
													Description: "Literal value mappings for the query.",
													ElementType: types.StringType,
												},
												"metric_value": schema.StringAttribute{
													Optional:    true,
													Description: "Metric value for the query.",
												},
											},
										},
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

func (r *customModelRulesResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data customModelRulesModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	rules, diags := modelToAPIRules(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	name := data.Name.ValueString()
	rules.Name = &name
	rules.SetManagedBy(getManagedByTerraformValue())

	stackID := fmt.Sprintf("%d", r.stackID)
	_, err := r.client.ModelRulesConfigurationAPI.PutModelRules(ctx).ModelRulesDto(*rules).XScopeOrgID(stackID).Execute()
	if err != nil {
		resp.Diagnostics.AddError("Failed to create custom model rules", err.Error())
		return
	}

	data.ID = types.StringValue(name)

	readData, diags := r.readModelWithRetry(ctx, name)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if readData == nil {
		resp.Diagnostics.AddError("Resource not found after create",
			fmt.Sprintf("custom model rules %q was not found after creation", name))
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, readData)...)
}

func (r *customModelRulesResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data customModelRulesModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	readData, diags := r.readModelWithRetry(ctx, data.ID.ValueString())
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if readData == nil {
		resp.State.RemoveResource(ctx)
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, readData)...)
}

func (r *customModelRulesResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data customModelRulesModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	rules, diags := modelToAPIRules(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	name := data.Name.ValueString()
	rules.Name = &name
	rules.SetManagedBy(getManagedByTerraformValue())

	stackID := fmt.Sprintf("%d", r.stackID)
	_, err := r.client.ModelRulesConfigurationAPI.PutModelRules(ctx).ModelRulesDto(*rules).XScopeOrgID(stackID).Execute()
	if err != nil {
		resp.Diagnostics.AddError("Failed to update custom model rules", err.Error())
		return
	}

	readData, diags := r.readModelWithRetry(ctx, name)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if readData == nil {
		resp.Diagnostics.AddError("Resource not found after update",
			fmt.Sprintf("custom model rules %q was not found after update", name))
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, readData)...)
}

func (r *customModelRulesResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data customModelRulesModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	name := data.ID.ValueString()
	stackID := fmt.Sprintf("%d", r.stackID)
	_, err := r.client.ModelRulesConfigurationAPI.DeleteModelRules(ctx, name).XScopeOrgID(stackID).Execute()
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete custom model rules", err.Error())
	}
}

func (r *customModelRulesResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	readData, diags := r.readModelWithRetry(ctx, req.ID)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if readData == nil {
		resp.Diagnostics.AddError("Resource not found",
			fmt.Sprintf("custom model rules %q not found during import", req.ID))
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, readData)...)
}

// readModelWithRetry fetches the custom model rules from the API with retry/backoff
// to handle eventual consistency after write operations.
func (r *customModelRulesResource) readModelWithRetry(ctx context.Context, name string) (*customModelRulesModel, diag.Diagnostics) {
	var diags diag.Diagnostics
	maxRetries := 40
	deadline := time.Now().Add(600 * time.Second)
	stackID := fmt.Sprintf("%d", r.stackID)
	var lastErrMsg string

	for attempt := 1; attempt <= maxRetries; attempt++ {
		if time.Now().After(deadline) {
			diags.AddError("Timeout reading custom model rules",
				fmt.Sprintf("timed out waiting for custom model rules %q after 600s", name))
			return nil, diags
		}
		select {
		case <-ctx.Done():
			diags.AddError("Context cancelled", ctx.Err().Error())
			return nil, diags
		default:
		}

		result, _, err := r.client.ModelRulesConfigurationAPI.GetModelRules(ctx, name).XScopeOrgID(stackID).Execute()
		if err == nil {
			if result == nil {
				return nil, diags
			}
			return apiRulesToModel(ctx, name, result)
		}

		if _, ok := err.(*assertsapi.GenericOpenAPIError); ok {
			lastErrMsg = fmt.Sprintf("custom model rules %q not found (attempt %d/%d)", name, attempt, maxRetries)
			if attempt >= maxRetries {
				diags.AddError("Custom model rules not found",
					fmt.Sprintf("giving up after %d attempt(s): %s", attempt, lastErrMsg))
				return nil, diags
			}
		} else {
			lastErrMsg = fmt.Sprintf("API error: %s", err)
			if attempt >= maxRetries {
				diags.AddError("Failed to read custom model rules",
					fmt.Sprintf("giving up after %d attempt(s): %s", attempt, lastErrMsg))
				return nil, diags
			}
		}

		customModelRulesBackoff(ctx, attempt)
	}

	diags.AddError("Failed to read custom model rules",
		fmt.Sprintf("giving up after %d attempt(s): %s", maxRetries, lastErrMsg))
	return nil, diags
}

func customModelRulesBackoff(ctx context.Context, attempt int) {
	var baseSleep time.Duration
	if attempt == 1 {
		baseSleep = 1 * time.Second
	} else {
		baseSleep = time.Duration(1<<int(math.Min(float64(attempt-2), 4))) * time.Second
	}
	minSleep := baseSleep / 2
	maxJitter := baseSleep - minSleep
	var sleepDuration time.Duration
	if maxJitter > 0 {
		//nolint:gosec
		j := time.Duration(rand.Int63n(int64(maxJitter)))
		sleepDuration = minSleep + j
	} else {
		sleepDuration = baseSleep
	}
	select {
	case <-ctx.Done():
	case <-time.After(sleepDuration):
	}
}

// apiRulesToModel converts an API ModelRulesDto to the Terraform Framework model.
func apiRulesToModel(ctx context.Context, name string, rules *assertsapi.ModelRulesDto) (*customModelRulesModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	if rules == nil {
		return nil, diags
	}

	modelName := name
	if rules.Name != nil {
		modelName = *rules.Name
	}

	model := &customModelRulesModel{
		ID:   types.StringValue(name),
		Name: types.StringValue(modelName),
	}

	var entities []entityModel
	for _, entity := range rules.Entities {
		em, d := apiEntityToModel(ctx, entity)
		diags.Append(d...)
		if diags.HasError() {
			return nil, diags
		}
		entities = append(entities, em)
	}
	if entities == nil {
		entities = []entityModel{}
	}
	model.Rules = []rulesModel{{Entity: entities}}
	return model, diags
}

func apiEntityToModel(ctx context.Context, entity assertsapi.EntityRuleDto) (entityModel, diag.Diagnostics) {
	var diags diag.Diagnostics
	em := entityModel{}

	if entity.Type != nil {
		em.Type = types.StringValue(*entity.Type)
	}
	if entity.Name != nil {
		em.Name = types.StringValue(*entity.Name)
	}

	if len(entity.Scope) > 0 {
		scopeMap, d := types.MapValueFrom(ctx, types.StringType, entity.Scope)
		diags.Append(d...)
		em.Scope = scopeMap
	} else {
		em.Scope = types.MapNull(types.StringType)
	}

	if len(entity.Lookup) > 0 {
		lookupMap, d := types.MapValueFrom(ctx, types.StringType, entity.Lookup)
		diags.Append(d...)
		em.Lookup = lookupMap
	} else {
		em.Lookup = types.MapNull(types.StringType)
	}

	if len(entity.EnrichedBy) > 0 {
		queries := make([]string, 0, len(entity.EnrichedBy))
		for _, eb := range entity.EnrichedBy {
			if eb.Query != nil {
				queries = append(queries, *eb.Query)
			}
		}
		enrichedByList, d := types.ListValueFrom(ctx, types.StringType, queries)
		diags.Append(d...)
		em.EnrichedBy = enrichedByList
	} else {
		em.EnrichedBy = types.ListNull(types.StringType)
	}

	if entity.Disabled != nil {
		em.Disabled = types.BoolValue(*entity.Disabled)
	} else {
		em.Disabled = types.BoolNull()
	}

	var definedBy []definedByModel
	for _, db := range entity.DefinedBy {
		dm, d := apiDefinedByToModel(ctx, db)
		diags.Append(d...)
		if diags.HasError() {
			return em, diags
		}
		definedBy = append(definedBy, dm)
	}
	if definedBy == nil {
		definedBy = []definedByModel{}
	}
	em.DefinedBy = definedBy

	return em, diags
}

func apiDefinedByToModel(ctx context.Context, db assertsapi.PropertyRuleDto) (definedByModel, diag.Diagnostics) {
	var diags diag.Diagnostics
	dm := definedByModel{}

	if db.Query != nil {
		dm.Query = types.StringValue(*db.Query)
	}

	if db.Disabled != nil {
		dm.Disabled = types.BoolValue(*db.Disabled)
	} else {
		dm.Disabled = types.BoolNull()
	}

	if len(db.LabelValues) > 0 {
		lvMap, d := types.MapValueFrom(ctx, types.StringType, db.LabelValues)
		diags.Append(d...)
		dm.LabelValues = lvMap
	} else {
		dm.LabelValues = types.MapNull(types.StringType)
	}

	if len(db.Literals) > 0 {
		litMap, d := types.MapValueFrom(ctx, types.StringType, db.Literals)
		diags.Append(d...)
		dm.Literals = litMap
	} else {
		dm.Literals = types.MapNull(types.StringType)
	}

	if db.MetricValue != nil && *db.MetricValue != "" {
		dm.MetricValue = types.StringValue(*db.MetricValue)
	} else {
		dm.MetricValue = types.StringNull()
	}

	return dm, diags
}

// modelToAPIRules converts the Terraform Framework model to an API ModelRulesDto.
func modelToAPIRules(ctx context.Context, data *customModelRulesModel) (*assertsapi.ModelRulesDto, diag.Diagnostics) {
	var diags diag.Diagnostics

	if len(data.Rules) == 0 {
		diags.AddError("rules block is required", "at least one rules block must be specified")
		return nil, diags
	}

	rulesData := data.Rules[0]
	var entities []assertsapi.EntityRuleDto
	for _, entityData := range rulesData.Entity {
		entity, d := modelEntityToAPI(ctx, entityData)
		diags.Append(d...)
		if diags.HasError() {
			return nil, diags
		}
		entities = append(entities, entity)
	}

	return &assertsapi.ModelRulesDto{Entities: entities}, diags
}

func modelEntityToAPI(ctx context.Context, em entityModel) (assertsapi.EntityRuleDto, diag.Diagnostics) {
	var diags diag.Diagnostics

	entityType := em.Type.ValueString()
	entityName := em.Name.ValueString()

	var definedBy []assertsapi.PropertyRuleDto
	for _, dm := range em.DefinedBy {
		prop, d := modelDefinedByToAPI(ctx, dm)
		diags.Append(d...)
		if diags.HasError() {
			return assertsapi.EntityRuleDto{}, diags
		}
		definedBy = append(definedBy, prop)
	}

	entity := assertsapi.EntityRuleDto{
		Type:      &entityType,
		Name:      &entityName,
		DefinedBy: definedBy,
	}

	if !em.Scope.IsNull() && !em.Scope.IsUnknown() {
		scopeMap := make(map[string]string)
		diags.Append(em.Scope.ElementsAs(ctx, &scopeMap, false)...)
		entity.Scope = scopeMap
	}

	if !em.Lookup.IsNull() && !em.Lookup.IsUnknown() {
		lookupMap := make(map[string]string)
		diags.Append(em.Lookup.ElementsAs(ctx, &lookupMap, false)...)
		entity.Lookup = lookupMap
	}

	if !em.EnrichedBy.IsNull() && !em.EnrichedBy.IsUnknown() {
		var queries []string
		diags.Append(em.EnrichedBy.ElementsAs(ctx, &queries, false)...)
		for _, q := range queries {
			qCopy := q
			entity.EnrichedBy = append(entity.EnrichedBy, assertsapi.PropertyRuleDto{Query: &qCopy})
		}
	}

	// Only set disabled when explicitly true, matching original behavior.
	if !em.Disabled.IsNull() && em.Disabled.ValueBool() {
		disabled := true
		entity.Disabled = &disabled
	}

	return entity, diags
}

func modelDefinedByToAPI(ctx context.Context, dm definedByModel) (assertsapi.PropertyRuleDto, diag.Diagnostics) {
	var diags diag.Diagnostics

	query := dm.Query.ValueString()
	prop := assertsapi.PropertyRuleDto{Query: &query}

	// Only set disabled when explicitly true, matching original behavior.
	if !dm.Disabled.IsNull() && dm.Disabled.ValueBool() {
		disabled := true
		prop.Disabled = &disabled
	}

	if !dm.LabelValues.IsNull() && !dm.LabelValues.IsUnknown() {
		labelMap := make(map[string]string)
		diags.Append(dm.LabelValues.ElementsAs(ctx, &labelMap, false)...)
		if len(labelMap) > 0 {
			prop.LabelValues = labelMap
		}
	}

	if !dm.Literals.IsNull() && !dm.Literals.IsUnknown() {
		litMap := make(map[string]string)
		diags.Append(dm.Literals.ElementsAs(ctx, &litMap, false)...)
		if len(litMap) > 0 {
			prop.Literals = litMap
		}
	}

	if !dm.MetricValue.IsNull() && !dm.MetricValue.IsUnknown() && dm.MetricValue.ValueString() != "" {
		mv := dm.MetricValue.ValueString()
		prop.MetricValue = &mv
	}

	return prop, diags
}
