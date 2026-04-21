package asserts

import (
	"context"
	"testing"

	assertsapi "github.com/grafana/grafana-asserts-public-clients/go/gcom"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testRulesName      = "test-rules"
	testEntityTypeServ = "Service"
)

func TestUnitCustomModelRules_Metadata(t *testing.T) {
	r := &customModelRulesResource{}
	var resp resource.MetadataResponse
	r.Metadata(context.Background(), resource.MetadataRequest{}, &resp)
	assert.Equal(t, "grafana_asserts_custom_model_rules", resp.TypeName)
}

func TestUnitCustomModelRules_Schema(t *testing.T) {
	r := &customModelRulesResource{}
	var resp resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &resp)
	require.False(t, resp.Diagnostics.HasError(), "schema should be valid: %v", resp.Diagnostics)

	attrs := resp.Schema.Attributes
	assert.Contains(t, attrs, "id")
	assert.Contains(t, attrs, "name")

	blocks := resp.Schema.Blocks
	assert.Contains(t, blocks, "rules")
	assert.NotNil(t, blocks["rules"])
}

// TestUnitCustomModelRules_APIToModel_Basic verifies that a minimal API response converts correctly.
func TestUnitCustomModelRules_APIToModel_Basic(t *testing.T) {
	ctx := context.Background()
	name := testRulesName
	query := "up{job!=''}"
	entityType := testEntityTypeServ
	entityName := testEntityTypeServ

	apiRules := &assertsapi.ModelRulesDto{
		Name: &name,
		Entities: []assertsapi.EntityRuleDto{
			{
				Type: &entityType,
				Name: &entityName,
				DefinedBy: []assertsapi.PropertyRuleDto{
					{Query: &query},
				},
			},
		},
	}

	model, diags := apiRulesToModel(ctx, name, apiRules)
	require.False(t, diags.HasError(), "unexpected diags: %v", diags)
	require.NotNil(t, model)

	assert.Equal(t, name, model.ID.ValueString())
	assert.Equal(t, name, model.Name.ValueString())
	require.Len(t, model.Rules, 1)
	require.Len(t, model.Rules[0].Entity, 1)

	entity := model.Rules[0].Entity[0]
	assert.Equal(t, entityType, entity.Type.ValueString())
	assert.Equal(t, entityName, entity.Name.ValueString())
	assert.True(t, entity.Scope.IsNull(), "absent scope should be null")
	assert.True(t, entity.Lookup.IsNull(), "absent lookup should be null")
	assert.True(t, entity.EnrichedBy.IsNull(), "absent enriched_by should be null")
	assert.Equal(t, types.BoolValue(false), entity.Disabled, "absent disabled should default to false")

	require.Len(t, entity.DefinedBy, 1)
	db := entity.DefinedBy[0]
	assert.Equal(t, query, db.Query.ValueString())
	assert.Equal(t, types.BoolValue(false), db.Disabled)
	assert.True(t, db.LabelValues.IsNull())
	assert.True(t, db.Literals.IsNull())
	assert.True(t, db.MetricValue.IsNull())
}

// TestUnitCustomModelRules_APIToModel_AllOptionalFields verifies optional fields convert correctly.
func TestUnitCustomModelRules_APIToModel_AllOptionalFields(t *testing.T) {
	ctx := context.Background()
	name := testRulesName
	query := "up{job!=''}"
	entityType := testEntityTypeServ
	entityName := "workload"
	enrichedQuery := "kube_pod_info"
	metricVal := "value"
	disabled := true

	apiRules := &assertsapi.ModelRulesDto{
		Name: &name,
		Entities: []assertsapi.EntityRuleDto{
			{
				Type:     &entityType,
				Name:     &entityName,
				Scope:    map[string]string{"namespace": "ns", "env": "prod"},
				Lookup:   map[string]string{"workload": "workload"},
				Disabled: &disabled,
				EnrichedBy: []assertsapi.PropertyRuleDto{
					{Query: &enrichedQuery},
				},
				DefinedBy: []assertsapi.PropertyRuleDto{
					{
						Query:       &query,
						Disabled:    &disabled,
						LabelValues: map[string]string{"job": "job"},
						Literals:    map[string]string{"_src": "test"},
						MetricValue: &metricVal,
					},
				},
			},
		},
	}

	model, diags := apiRulesToModel(ctx, name, apiRules)
	require.False(t, diags.HasError())
	require.NotNil(t, model)

	entity := model.Rules[0].Entity[0]

	// scope map
	require.False(t, entity.Scope.IsNull())
	scopeElems := entity.Scope.Elements()
	assert.Len(t, scopeElems, 2)
	assert.Equal(t, types.StringValue("ns"), scopeElems["namespace"])

	// lookup map
	require.False(t, entity.Lookup.IsNull())
	lookupElems := entity.Lookup.Elements()
	assert.Equal(t, types.StringValue("workload"), lookupElems["workload"])

	// enriched_by list
	require.False(t, entity.EnrichedBy.IsNull())
	enrichedElems := entity.EnrichedBy.Elements()
	assert.Len(t, enrichedElems, 1)
	assert.Equal(t, types.StringValue(enrichedQuery), enrichedElems[0])

	// entity disabled
	assert.Equal(t, types.BoolValue(true), entity.Disabled)

	// defined_by fields
	db := entity.DefinedBy[0]
	assert.Equal(t, types.BoolValue(true), db.Disabled)
	assert.Equal(t, types.StringValue(metricVal), db.MetricValue)

	lvElems := db.LabelValues.Elements()
	assert.Equal(t, types.StringValue("job"), lvElems["job"])
	litElems := db.Literals.Elements()
	assert.Equal(t, types.StringValue("test"), litElems["_src"])
}

// TestUnitCustomModelRules_APIToModel_NilInput verifies nil API response returns nil model.
func TestUnitCustomModelRules_APIToModel_NilInput(t *testing.T) {
	model, diags := apiRulesToModel(context.Background(), "name", nil)
	require.False(t, diags.HasError())
	assert.Nil(t, model)
}

// TestUnitCustomModelRules_ModelToAPI_Basic verifies a minimal model converts correctly.
func TestUnitCustomModelRules_ModelToAPI_Basic(t *testing.T) {
	ctx := context.Background()
	data := &customModelRulesModel{
		ID:   types.StringValue(testRulesName),
		Name: types.StringValue(testRulesName),
		Rules: []rulesModel{
			{
				Entity: []entityModel{
					{
						Type:       types.StringValue(testEntityTypeServ),
						Name:       types.StringValue(testEntityTypeServ),
						Scope:      types.MapNull(types.StringType),
						Lookup:     types.MapNull(types.StringType),
						EnrichedBy: types.ListNull(types.StringType),
						Disabled:   types.BoolNull(),
						DefinedBy: []definedByModel{
							{
								Query:       types.StringValue("up{job!=''}"),
								Disabled:    types.BoolNull(),
								LabelValues: types.MapNull(types.StringType),
								Literals:    types.MapNull(types.StringType),
								MetricValue: types.StringNull(),
							},
						},
					},
				},
			},
		},
	}

	apiRules, diags := modelToAPIRules(ctx, data)
	require.False(t, diags.HasError())
	require.NotNil(t, apiRules)
	require.Len(t, apiRules.Entities, 1)

	entity := apiRules.Entities[0]
	assert.Equal(t, testEntityTypeServ, *entity.Type)
	assert.Equal(t, testEntityTypeServ, *entity.Name)
	assert.Nil(t, entity.Scope)
	assert.Nil(t, entity.Lookup)
	assert.Nil(t, entity.EnrichedBy)
	assert.Nil(t, entity.Disabled)
	require.Len(t, entity.DefinedBy, 1)
	assert.Equal(t, "up{job!=''}", *entity.DefinedBy[0].Query)
	assert.Nil(t, entity.DefinedBy[0].Disabled)
}

// TestUnitCustomModelRules_ModelToAPI_DisabledOnlySetWhenTrue verifies the original behavior
// that disabled is only sent to the API when explicitly true (not when false).
func TestUnitCustomModelRules_ModelToAPI_DisabledOnlySetWhenTrue(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name            string
		entityDisabled  types.Bool
		definedDisabled types.Bool
		wantEntityNil   bool
		wantDBNil       bool
	}{
		{
			name:            "null disabled → nil in API",
			entityDisabled:  types.BoolNull(),
			definedDisabled: types.BoolNull(),
			wantEntityNil:   true,
			wantDBNil:       true,
		},
		{
			name:            "false disabled → nil in API",
			entityDisabled:  types.BoolValue(false),
			definedDisabled: types.BoolValue(false),
			wantEntityNil:   true,
			wantDBNil:       true,
		},
		{
			name:            "true disabled → set in API",
			entityDisabled:  types.BoolValue(true),
			definedDisabled: types.BoolValue(true),
			wantEntityNil:   false,
			wantDBNil:       false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			data := &customModelRulesModel{
				ID:   types.StringValue("r"),
				Name: types.StringValue("r"),
				Rules: []rulesModel{{Entity: []entityModel{{
					Type:       types.StringValue(testEntityTypeServ),
					Name:       types.StringValue(testEntityTypeServ),
					Scope:      types.MapNull(types.StringType),
					Lookup:     types.MapNull(types.StringType),
					EnrichedBy: types.ListNull(types.StringType),
					Disabled:   tc.entityDisabled,
					DefinedBy: []definedByModel{{
						Query:       types.StringValue("up{}"),
						Disabled:    tc.definedDisabled,
						LabelValues: types.MapNull(types.StringType),
						Literals:    types.MapNull(types.StringType),
						MetricValue: types.StringNull(),
					}},
				}}}},
			}

			apiRules, diags := modelToAPIRules(ctx, data)
			require.False(t, diags.HasError())

			entity := apiRules.Entities[0]
			if tc.wantEntityNil {
				assert.Nil(t, entity.Disabled)
			} else {
				require.NotNil(t, entity.Disabled)
				assert.True(t, *entity.Disabled)
			}

			db := entity.DefinedBy[0]
			if tc.wantDBNil {
				assert.Nil(t, db.Disabled)
			} else {
				require.NotNil(t, db.Disabled)
				assert.True(t, *db.Disabled)
			}
		})
	}
}

// TestUnitCustomModelRules_RoundTrip verifies that API→model→API produces the same API payload.
func TestUnitCustomModelRules_RoundTrip(t *testing.T) {
	ctx := context.Background()
	name := testRulesName
	query := "up{job!=''}"
	entityType := testEntityTypeServ
	entityName := "workload"
	enrichedQuery := "kube_pod_info"
	disabled := true

	originalAPI := &assertsapi.ModelRulesDto{
		Name: &name,
		Entities: []assertsapi.EntityRuleDto{
			{
				Type:     &entityType,
				Name:     &entityName,
				Scope:    map[string]string{"namespace": "ns"},
				Lookup:   map[string]string{"workload": "workload"},
				Disabled: &disabled,
				EnrichedBy: []assertsapi.PropertyRuleDto{
					{Query: &enrichedQuery},
				},
				DefinedBy: []assertsapi.PropertyRuleDto{
					{
						Query:       &query,
						LabelValues: map[string]string{"job": "job"},
						Literals:    map[string]string{"_src": "test"},
					},
				},
			},
		},
	}

	// API → model
	model, diags := apiRulesToModel(ctx, name, originalAPI)
	require.False(t, diags.HasError())
	require.NotNil(t, model)

	// model → API
	roundTripped, diags := modelToAPIRules(ctx, model)
	require.False(t, diags.HasError())
	require.NotNil(t, roundTripped)

	require.Len(t, roundTripped.Entities, 1)
	entity := roundTripped.Entities[0]

	assert.Equal(t, entityType, *entity.Type)
	assert.Equal(t, entityName, *entity.Name)
	assert.Equal(t, map[string]string{"namespace": "ns"}, entity.Scope)
	assert.Equal(t, map[string]string{"workload": "workload"}, entity.Lookup)
	require.NotNil(t, entity.Disabled)
	assert.True(t, *entity.Disabled)

	require.Len(t, entity.EnrichedBy, 1)
	assert.Equal(t, enrichedQuery, *entity.EnrichedBy[0].Query)

	require.Len(t, entity.DefinedBy, 1)
	db := entity.DefinedBy[0]
	assert.Equal(t, query, *db.Query)
	assert.Equal(t, map[string]string{"job": "job"}, db.LabelValues)
	assert.Equal(t, map[string]string{"_src": "test"}, db.Literals)
}

// TestUnitCustomModelRules_RoundTrip_WithDisabled verifies the disabled field round-trips correctly.
func TestUnitCustomModelRules_RoundTrip_WithDisabled(t *testing.T) {
	ctx := context.Background()
	name := testRulesName
	query := "up{}"
	entityType := testEntityTypeServ
	entityName := testEntityTypeServ
	disabled := true

	originalAPI := &assertsapi.ModelRulesDto{
		Name: &name,
		Entities: []assertsapi.EntityRuleDto{
			{
				Type:     &entityType,
				Name:     &entityName,
				Disabled: &disabled,
				DefinedBy: []assertsapi.PropertyRuleDto{
					{Query: &query, Disabled: &disabled},
				},
			},
		},
	}

	model, diags := apiRulesToModel(ctx, name, originalAPI)
	require.False(t, diags.HasError())

	roundTripped, diags := modelToAPIRules(ctx, model)
	require.False(t, diags.HasError())

	entity := roundTripped.Entities[0]
	require.NotNil(t, entity.Disabled)
	assert.True(t, *entity.Disabled)

	db := entity.DefinedBy[0]
	require.NotNil(t, db.Disabled)
	assert.True(t, *db.Disabled)
}

// TestUnitCustomModelRules_EmptyEntities verifies an empty entities list produces a valid model.
func TestUnitCustomModelRules_EmptyEntities(t *testing.T) {
	ctx := context.Background()
	name := testRulesName

	apiRules := &assertsapi.ModelRulesDto{
		Name:     &name,
		Entities: []assertsapi.EntityRuleDto{},
	}

	model, diags := apiRulesToModel(ctx, name, apiRules)
	require.False(t, diags.HasError())
	require.NotNil(t, model)
	require.Len(t, model.Rules, 1)
	assert.Empty(t, model.Rules[0].Entity)
}

// TestUnitCustomModelRules_ModelToAPI_EmptyRules verifies error on empty rules.
func TestUnitCustomModelRules_ModelToAPI_EmptyRules(t *testing.T) {
	data := &customModelRulesModel{
		ID:    types.StringValue("r"),
		Name:  types.StringValue("r"),
		Rules: []rulesModel{},
	}
	_, diags := modelToAPIRules(context.Background(), data)
	assert.True(t, diags.HasError())
}

// TestUnitCustomModelRules_ModelToAPI_EmptyCollectionsOmitted verifies that empty (non-null) scope,
// lookup, and enriched_by are not sent to the API. The read-path normalises absent API fields back to
// null, so sending an empty collection would cause a perpetual null-vs-empty plan diff.
func TestUnitCustomModelRules_ModelToAPI_EmptyCollectionsOmitted(t *testing.T) {
	ctx := context.Background()

	emptyScope, _ := types.MapValueFrom(ctx, types.StringType, map[string]string{})
	emptyLookup, _ := types.MapValueFrom(ctx, types.StringType, map[string]string{})
	emptyEnrichedBy, _ := types.ListValueFrom(ctx, types.StringType, []string{})

	data := &customModelRulesModel{
		ID:   types.StringValue("r"),
		Name: types.StringValue("r"),
		Rules: []rulesModel{{Entity: []entityModel{{
			Type:       types.StringValue(testEntityTypeServ),
			Name:       types.StringValue(testEntityTypeServ),
			Scope:      emptyScope,
			Lookup:     emptyLookup,
			EnrichedBy: emptyEnrichedBy,
			Disabled:   types.BoolNull(),
			DefinedBy: []definedByModel{{
				Query:       types.StringValue("up{}"),
				Disabled:    types.BoolNull(),
				LabelValues: types.MapNull(types.StringType),
				Literals:    types.MapNull(types.StringType),
				MetricValue: types.StringNull(),
			}},
		}}}},
	}

	apiRules, diags := modelToAPIRules(ctx, data)
	require.False(t, diags.HasError())

	entity := apiRules.Entities[0]
	assert.Nil(t, entity.Scope, "empty scope must not be sent to avoid null-vs-empty diff")
	assert.Nil(t, entity.Lookup, "empty lookup must not be sent to avoid null-vs-empty diff")
	assert.Nil(t, entity.EnrichedBy, "empty enriched_by must not be sent to avoid null-vs-empty diff")
}

// TestUnitCustomModelRules_Configure_MissingStackID verifies that Configure rejects a zero stack_id.
func TestUnitCustomModelRules_Configure_MissingStackID(t *testing.T) {
	r := &customModelRulesResource{}
	req := resource.ConfigureRequest{
		ProviderData: &common.Client{
			AssertsAPIClient: assertsapi.NewAPIClient(assertsapi.NewConfiguration()),
			GrafanaStackID:   0,
		},
	}
	var resp resource.ConfigureResponse
	r.Configure(context.Background(), req, &resp)

	require.True(t, resp.Diagnostics.HasError())
	assert.Contains(t, resp.Diagnostics.Errors()[0].Summary(), "stack_id")
	assert.Nil(t, r.client, "client must not be set when stack_id is missing")
}

// TestUnitCustomModelRules_Configure_MissingClient verifies that Configure rejects a nil AssertsAPIClient.
func TestUnitCustomModelRules_Configure_MissingClient(t *testing.T) {
	r := &customModelRulesResource{}
	req := resource.ConfigureRequest{
		ProviderData: &common.Client{
			AssertsAPIClient: nil,
			GrafanaStackID:   42,
		},
	}
	var resp resource.ConfigureResponse
	r.Configure(context.Background(), req, &resp)

	require.True(t, resp.Diagnostics.HasError())
	assert.Nil(t, r.client)
}

// TestUnitCustomModelRules_Configure_Valid verifies that Configure succeeds with both fields set.
func TestUnitCustomModelRules_Configure_Valid(t *testing.T) {
	r := &customModelRulesResource{}
	apiClient := assertsapi.NewAPIClient(assertsapi.NewConfiguration())
	req := resource.ConfigureRequest{
		ProviderData: &common.Client{
			AssertsAPIClient: apiClient,
			GrafanaStackID:   99,
		},
	}
	var resp resource.ConfigureResponse
	r.Configure(context.Background(), req, &resp)

	require.False(t, resp.Diagnostics.HasError())
	assert.Equal(t, apiClient, r.client)
	assert.Equal(t, int64(99), r.stackID)
}

// TestUnitCustomModelRules_IsRequired_NullBlock verifies that IsRequired() fires on a null block,
// TestUnitCustomModelRules_DisabledFalse_APIReturnsNil verifies that when the API omits the disabled
// field (returns nil), the read path stores false rather than null. This prevents a perpetual plan
// diff when config sets disabled = false: apply sends nothing, API returns nil, state must match.
func TestUnitCustomModelRules_DisabledFalse_APIReturnsNil(t *testing.T) {
	ctx := context.Background()
	name := testRulesName
	query := "up{}"
	entityType := testEntityTypeServ
	entityName := testEntityTypeServ

	// API response omits Disabled entirely (nil pointer).
	apiRules := &assertsapi.ModelRulesDto{
		Name: &name,
		Entities: []assertsapi.EntityRuleDto{
			{
				Type:     &entityType,
				Name:     &entityName,
				Disabled: nil,
				DefinedBy: []assertsapi.PropertyRuleDto{
					{Query: &query, Disabled: nil},
				},
			},
		},
	}

	model, diags := apiRulesToModel(ctx, name, apiRules)
	require.False(t, diags.HasError())
	require.Len(t, model.Rules, 1)
	require.Len(t, model.Rules[0].Entity, 1)

	entity := model.Rules[0].Entity[0]
	assert.Equal(t, types.BoolValue(false), entity.Disabled, "entity.Disabled must be false (not null) when API omits the field")

	require.Len(t, entity.DefinedBy, 1)
	assert.Equal(t, types.BoolValue(false), entity.DefinedBy[0].Disabled, "defined_by.Disabled must be false (not null) when API omits the field")
}

// TestUnitCustomModelRules_DisabledFalse_RoundTrip verifies idempotency when disabled=false is in config:
// model→API does NOT send the field (omit-when-false), API returns nil, read path produces false.
// After a second apply the model and API are consistent.
func TestUnitCustomModelRules_DisabledFalse_RoundTrip(t *testing.T) {
	ctx := context.Background()
	disabledFalse := false

	// Simulate a model with disabled explicitly set to false (typical config state).
	data := &customModelRulesModel{
		ID:   types.StringValue(testRulesName),
		Name: types.StringValue(testRulesName),
		Rules: []rulesModel{{Entity: []entityModel{{
			Type:      types.StringValue(testEntityTypeServ),
			Name:      types.StringValue(testEntityTypeServ),
			Scope:     types.MapNull(types.StringType),
			Lookup:    types.MapNull(types.StringType),
			EnrichedBy: types.ListNull(types.StringType),
			Disabled:  types.BoolValue(false),
			DefinedBy: []definedByModel{{
				Query:       types.StringValue("up{}"),
				Disabled:    types.BoolValue(false),
				LabelValues: types.MapNull(types.StringType),
				Literals:    types.MapNull(types.StringType),
				MetricValue: types.StringNull(),
			}},
		}}}},
	}

	apiPayload, diags := modelToAPIRules(ctx, data)
	require.False(t, diags.HasError())

	entity := apiPayload.Entities[0]
	// disabled=false must NOT be sent to the API (API uses omit-zero semantics).
	assert.Nil(t, entity.Disabled, "disabled=false must be omitted from the API payload")
	assert.Nil(t, entity.DefinedBy[0].Disabled, "defined_by.disabled=false must be omitted from the API payload")

	// Now simulate the API returning nil for disabled (typical API behaviour for false).
	entity.Disabled = nil
	entity.DefinedBy[0].Disabled = nil
	apiPayload.Entities[0] = entity

	// Read the API response back into a model.
	apiRules := &assertsapi.ModelRulesDto{
		Name:     &[]string{testRulesName}[0],
		Entities: apiPayload.Entities,
	}
	readModel, diags := apiRulesToModel(ctx, testRulesName, apiRules)
	require.False(t, diags.HasError())

	readEntity := readModel.Rules[0].Entity[0]
	// State must record false, matching the original config value — no perpetual diff.
	assert.Equal(t, types.BoolValue(false), readEntity.Disabled)
	assert.Equal(t, types.BoolValue(false), readEntity.DefinedBy[0].Disabled)

	// Verify the original model and the round-tripped model agree on disabled.
	originalDisabled := disabledFalse
	_ = originalDisabled // false, matches what the read path now returns
}

// which SizeAtLeast(1) would silently skip.
func TestUnitCustomModelRules_IsRequired_NullBlock(t *testing.T) {
	ctx := context.Background()
	v := listvalidator.IsRequired()

	// Null list (block completely absent from config) must produce an error.
	nullReq := validator.ListRequest{
		Path:        path.Root("rules"),
		ConfigValue: types.ListNull(types.ObjectType{}),
	}
	var nullResp validator.ListResponse
	v.ValidateList(ctx, nullReq, &nullResp)
	assert.True(t, nullResp.Diagnostics.HasError(), "IsRequired must error on null block")

	// Empty (but non-null) list must pass IsRequired — SizeAtLeast catches the empty case.
	emptyReq := validator.ListRequest{
		Path:        path.Root("rules"),
		ConfigValue: types.ListValueMust(types.ObjectType{}, []attr.Value{}),
	}
	var emptyResp validator.ListResponse
	v.ValidateList(ctx, emptyReq, &emptyResp)
	assert.False(t, emptyResp.Diagnostics.HasError(), "IsRequired must pass on empty (non-null) list")
}
