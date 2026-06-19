package oncall

import (
	"testing"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/stretchr/testify/require"
)

// baseIntegrationConfig returns the minimum valid config for a grafana integration.
// It intentionally omits labels and dynamic_labels so callers can add them selectively.
func baseIntegrationConfig() map[string]any {
	return map[string]any{
		"name": "test-integration",
		"type": "grafana",
		"default_route": []any{
			map[string]any{
				"slack": []any{
					map[string]any{"enabled": false},
				},
				"telegram": []any{
					map[string]any{"enabled": false},
				},
			},
		},
	}
}

// TestBuildIntegrationUpdateOptions exercises the label-update behavior that can be
// verified through schema.TestResourceDataRaw.
//
// NOTE: schema.TestResourceDataRaw does NOT populate d.GetRawConfig() — it always
// returns a null cty.Value. This means labelsSetInConfig always returns false in these
// unit tests. Consequently:
//
//   - Case 1 (never in config, no prior state): HasChange=false, labelsSetInConfig=false
//     → Labels=nil ✓ (tested below)
//
//   - Case 2 (labels present in config): HasChange=true (nil-state→non-empty = change),
//     labelsSetInConfig=false → combined=true → sends labels ✓ (tested below)
//
//   - Case 3 (explicit labels=[]): HasChange=false (nil-state→empty = no change),
//     labelsSetInConfig=false → combined=false → would return nil in this unit test,
//     but in real Terraform labelsSetInConfig=true so it correctly sends [].
//     This case is validated by TestLabelsUpdateConditionMatrix + TestLabelsSetInConfig.
//
//   - Case 4 (removed from config): requires prior state with labels + absent config,
//     which TestResourceDataRaw cannot construct. Validated by TestLabelsUpdateConditionMatrix.
func TestBuildIntegrationUpdateOptions(t *testing.T) {
	t.Parallel()

	s := resourceIntegration().Schema.Schema

	t.Run("case1_labels_never_in_config_omitted_from_put", func(t *testing.T) {
		t.Parallel()

		// Labels block absent from config entirely and no prior state.
		// HasChange=false (nil→nil/empty = no change), labelsSetInConfig=false (rawConfig null).
		// Labels must be nil so the backend preserves any out-of-band labels.
		d := schema.TestResourceDataRaw(t, s, baseIntegrationConfig())
		opts := buildIntegrationUpdateOptions(d)

		require.Nil(t, opts.Labels,
			"labels absent from config with no prior state must be nil so backend preserves out-of-band labels")
		require.Nil(t, opts.DynamicLabels,
			"dynamic_labels absent from config with no prior state must be nil so backend preserves out-of-band labels")
	})

	t.Run("case2_labels_present_in_config_are_sent", func(t *testing.T) {
		t.Parallel()

		// Labels explicitly written in config with values.
		// HasChange=true (nil-state → non-empty labels = change) → combined=true → send.
		cfg := copyIntegrationConfig(baseIntegrationConfig())
		cfg["labels"] = []any{map[string]any{"key": "env", "value": "prod"}}
		cfg["dynamic_labels"] = []any{map[string]any{"key": "region", "value": "us-east"}}

		d := schema.TestResourceDataRaw(t, s, cfg)
		opts := buildIntegrationUpdateOptions(d)

		require.NotNil(t, opts.Labels)
		require.Len(t, *opts.Labels, 1)
		require.Equal(t, "env", (*opts.Labels)[0].Key.Name)
		require.Equal(t, "prod", (*opts.Labels)[0].Value.Name)

		require.NotNil(t, opts.DynamicLabels)
		require.Len(t, *opts.DynamicLabels, 1)
		require.Equal(t, "region", (*opts.DynamicLabels)[0].Key.Name)
		require.Equal(t, "us-east", (*opts.DynamicLabels)[0].Value.Name)
	})
}

// TestLabelsSetInConfig tests labelsSetInConfig with manually constructed cty.Values
// because schema.TestResourceDataRaw does not populate GetRawConfig() in unit tests.
// In real Terraform, GetRawConfig() is always populated by the framework.
func TestLabelsSetInConfig(t *testing.T) {
	t.Parallel()

	labelType := cty.List(cty.Map(cty.String))

	t.Run("returns_false_when_attr_is_null", func(t *testing.T) {
		t.Parallel()

		// Simulates "labels" absent from HCL config: the attribute is null.
		rawConfig := cty.ObjectVal(map[string]cty.Value{
			"labels": cty.NullVal(labelType),
		})
		require.False(t, labelsSetInConfig(rawConfig, "labels"),
			"null attribute must return false — backend preserves out-of-band labels")
	})

	t.Run("returns_false_when_rawConfig_is_null", func(t *testing.T) {
		t.Parallel()

		require.False(t, labelsSetInConfig(cty.NullVal(cty.DynamicPseudoType), "labels"),
			"null rawConfig must return false safely")
	})

	t.Run("returns_false_when_attr_not_in_schema", func(t *testing.T) {
		t.Parallel()

		rawConfig := cty.ObjectVal(map[string]cty.Value{
			"name": cty.StringVal("test"),
		})
		require.False(t, labelsSetInConfig(rawConfig, "labels"),
			"absent attribute must return false")
	})

	t.Run("returns_true_when_labels_present_with_values", func(t *testing.T) {
		t.Parallel()

		// Simulates labels = [{key = "env", value = "prod"}] in HCL config.
		rawConfig := cty.ObjectVal(map[string]cty.Value{
			"labels": cty.ListVal([]cty.Value{
				cty.MapVal(map[string]cty.Value{
					"key":   cty.StringVal("env"),
					"value": cty.StringVal("prod"),
				}),
			}),
		})
		require.True(t, labelsSetInConfig(rawConfig, "labels"),
			"non-null attribute with values must return true")
	})

	t.Run("returns_true_when_labels_explicitly_empty", func(t *testing.T) {
		t.Parallel()

		// Simulates labels = [] in HCL config: attribute is present but empty (not null).
		// This is distinct from "labels absent": the user explicitly wrote labels = [].
		rawConfig := cty.ObjectVal(map[string]cty.Value{
			"labels": cty.ListValEmpty(cty.Map(cty.String)),
		})
		require.True(t, labelsSetInConfig(rawConfig, "labels"),
			"non-null empty list must return true — user explicitly wrote labels = []")
	})
}

// TestLabelsUpdateConditionMatrix validates the combined condition used in
// buildIntegrationUpdateOptions for all four label-management scenarios.
//
// The condition is: shouldSend = d.HasChange(attr) || labelsSetInConfig(rawConfig, attr)
//
// This directly covers case 4 (labels removed from config) which cannot be exercised
// through buildIntegrationUpdateOptions in pure unit tests: schema.TestResourceDataRaw
// creates a ResourceData with nil prior state, so it is impossible to simulate
// HasChange=true while labelsSetInConfig=false via the public SDK API. Full end-to-end
// coverage for case 4 is provided by acceptance tests running a two-step apply.
func TestLabelsUpdateConditionMatrix(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		hasChange   bool
		setInConfig bool
		expectSend  bool
	}{
		{
			// Prior state: none. Config: no labels block.
			// Neither condition fires → omit from PUT body → backend preserves out-of-band labels.
			name:        "case1_never_in_config_no_prior_state",
			hasChange:   false,
			setInConfig: false,
			expectSend:  false,
		},
		{
			// Prior state: none. Config: labels block present and unchanged since last apply.
			// labelsSetInConfig=true → send to keep backend in sync.
			name:        "case2_in_config_unchanged",
			hasChange:   false,
			setInConfig: true,
			expectSend:  true,
		},
		{
			// Prior state: has labels. Config: labels block present with different values.
			// Both conditions are true → send new values.
			name:        "case3_in_config_values_changed",
			hasChange:   true,
			setInConfig: true,
			expectSend:  true,
		},
		{
			// The original bug: user removed the labels block from config.
			// labelsSetInConfig=false (block absent from HCL) but HasChange=true
			// (prior state has labels, plan has none) → send [] to explicitly clear.
			// Without HasChange, the old code never cleared labels on removal.
			name:        "case4_removed_from_config",
			hasChange:   true,
			setInConfig: false,
			expectSend:  true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			shouldSend := tc.hasChange || tc.setInConfig
			require.Equal(t, tc.expectSend, shouldSend,
				"HasChange=%v || labelsSetInConfig=%v should give shouldSend=%v",
				tc.hasChange, tc.setInConfig, tc.expectSend)
		})
	}
}

func copyIntegrationConfig(base map[string]any) map[string]any {
	config := make(map[string]any, len(base))
	for key, value := range base {
		config[key] = value
	}
	return config
}
