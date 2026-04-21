package asserts

import (
	"context"
	"testing"

	assertsapi "github.com/grafana/grafana-asserts-public-clients/go/gcom"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	name := "test-rules"
	query := "up{job!=''}"
	entityType := "Service"
	entityName := "Service"

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
	assert.True(t, entity.Disabled.IsNull(), "absent disabled should be null")

	require.Len(t, entity.DefinedBy, 1)
	db := entity.DefinedBy[0]
	assert.Equal(t, query, db.Query.ValueString())
	assert.True(t, db.Disabled.IsNull())
	assert.True(t, db.LabelValues.IsNull())
	assert.True(t, db.Literals.IsNull())
	assert.True(t, db.MetricValue.IsNull())
}

// TestUnitCustomModelRules_APIToModel_AllOptionalFields verifies optional fields convert correctly.
func TestUnitCustomModelRules_APIToModel_AllOptionalFields(t *testing.T) {
	ctx := context.Background()
	name := "test-rules"
	query := "up{job!=''}"
	entityType := "Service"
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
		ID:   types.StringValue("test-rules"),
		Name: types.StringValue("test-rules"),
		Rules: []rulesModel{
			{
				Entity: []entityModel{
					{
						Type:       types.StringValue("Service"),
						Name:       types.StringValue("Service"),
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
	assert.Equal(t, "Service", *entity.Type)
	assert.Equal(t, "Service", *entity.Name)
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
					Type:       types.StringValue("Service"),
					Name:       types.StringValue("Service"),
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
	name := "test-rules"
	query := "up{job!=''}"
	entityType := "Service"
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
	name := "test-rules"
	query := "up{}"
	entityType := "Service"
	entityName := "Service"
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
	name := "test-rules"

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
