package oncall

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/stretchr/testify/require"
	"github.com/hashicorp/go-cty/cty"
)

func TestIntegrationLabelsUpdateBehavior(t *testing.T) {
	t.Parallel()

	schemaMap := resourceIntegration().Schema.Schema
	baseConfig := map[string]any{
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
	label := map[string]any{"key": "TestKey", "value": "TestValue"}
	dynamicLabel := map[string]any{"key": "DynKey", "value": "DynValue"}

	t.Run("step1_create_with_labels", func(t *testing.T) {
		t.Parallel()

		config := copyIntegrationConfig(baseConfig)
		config["labels"] = []any{label}
		config["dynamic_labels"] = []any{dynamicLabel}

		d := schema.TestResourceDataRaw(t, schemaMap, config)

		labels := expandLabels(d.Get("labels").([]any))
		dynamicLabels := expandLabels(d.Get("dynamic_labels").([]any))

		require.Len(t, labels, 1)
		require.Equal(t, "TestKey", labels[0].Key.Name)
		require.Equal(t, "TestValue", labels[0].Value.Name)
		require.Len(t, dynamicLabels, 1)
		require.Equal(t, "DynKey", dynamicLabels[0].Key.Name)
		require.Equal(t, "DynValue", dynamicLabels[0].Value.Name)
	})

	t.Run("step2_update_omit_labels_preserves_existing", func(t *testing.T) {
		t.Parallel()

		configWithoutLabels := cty.ObjectVal(map[string]cty.Value{
			"name": cty.StringVal("test-integration"),
			"type": cty.StringVal("grafana"),
		})

		require.False(t, labelsSetInConfig(configWithoutLabels, "labels"))
		require.False(t, labelsSetInConfig(configWithoutLabels, "dynamic_labels"))

		d := schema.TestResourceDataRaw(t, schemaMap, copyIntegrationConfig(baseConfig))
		updateOptions := buildIntegrationUpdateOptions(d)

		require.Nil(t, updateOptions.Labels, "omitted labels should not be sent to preserve existing labels on the server")
		require.Nil(t, updateOptions.DynamicLabels, "omitted dynamic_labels should not be sent to preserve existing labels on the server")
	})

	t.Run("step3_update_empty_lists_clear_labels", func(t *testing.T) {
		t.Parallel()

		configWithEmptyLabels := cty.ObjectVal(map[string]cty.Value{
			"labels":         cty.ListValEmpty(cty.Map(cty.String)),
			"dynamic_labels": cty.ListValEmpty(cty.Map(cty.String)),
		})

		require.True(t, labelsSetInConfig(configWithEmptyLabels, "labels"))
		require.True(t, labelsSetInConfig(configWithEmptyLabels, "dynamic_labels"))

		labels := integrationUpdateLabelPointer(true, []any{})
		dynamicLabels := integrationUpdateLabelPointer(true, []any{})

		require.NotNil(t, labels)
		require.NotNil(t, dynamicLabels)
		require.Empty(t, *labels)
		require.Empty(t, *dynamicLabels)
	})
}

func copyIntegrationConfig(base map[string]any) map[string]any {
	config := make(map[string]any, len(base))
	for key, value := range base {
		config[key] = value
	}
	return config
}
