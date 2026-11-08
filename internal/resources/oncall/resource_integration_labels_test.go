package oncall

import (
	"strings"
	"testing"

	onCallAPI "github.com/grafana/amixr-api-go-client"
	"github.com/hashicorp/go-cty/cty"
)

func TestFlattenLabels_NoIDField(t *testing.T) {
	labels := []*onCallAPI.Label{
		{Key: onCallAPI.KeyValueName{Name: "severity"}, Value: onCallAPI.KeyValueName{Name: "critical"}},
	}

	result := flattenLabels(labels)

	if len(result) != 1 {
		t.Fatalf("expected 1 label, got %d", len(result))
	}
	if _, exists := result[0]["id"]; exists {
		t.Error("flattenLabels should not include an 'id' field; it causes spurious diffs because the config never specifies it")
	}
	if result[0]["key"] != "severity" {
		t.Errorf("expected key 'severity', got %q", result[0]["key"])
	}
	if result[0]["value"] != "critical" {
		t.Errorf("expected value 'critical', got %q", result[0]["value"])
	}
}

func TestFlattenLabels_Empty(t *testing.T) {
	result := flattenLabels(nil)
	if len(result) != 0 {
		t.Fatalf("expected 0 labels, got %d", len(result))
	}
}

func TestExpandLabels_NilElement(t *testing.T) {
	input := []any{
		nil,
		map[string]any{"key": "severity", "value": "critical"},
		nil,
	}

	result := expandLabels(input)

	if len(result) != 1 {
		t.Fatalf("expected 1 label, got %d", len(result))
	}
	if result[0].Key.Name != "severity" {
		t.Errorf("expected key 'severity', got %q", result[0].Key.Name)
	}
	if result[0].Value.Name != "critical" {
		t.Errorf("expected value 'critical', got %q", result[0].Value.Name)
	}
}

func TestExpandLabels_AllNil(t *testing.T) {
	input := []any{nil, nil}

	result := expandLabels(input)

	if len(result) != 0 {
		t.Fatalf("expected 0 labels, got %d", len(result))
	}
}

func TestLabelsSetEqual_SameOrder(t *testing.T) {
	a := []any{
		map[string]any{"key": "severity", "value": "critical"},
		map[string]any{"key": "team", "value": "platform"},
	}
	b := []any{
		map[string]any{"key": "severity", "value": "critical"},
		map[string]any{"key": "team", "value": "platform"},
	}
	if !labelsSetEqual(a, b) {
		t.Error("expected labels with same content and order to be equal")
	}
}

func TestLabelsSetEqual_DifferentOrder(t *testing.T) {
	a := []any{
		map[string]any{"key": "team", "value": "platform"},
		map[string]any{"key": "severity", "value": "critical"},
	}
	b := []any{
		map[string]any{"key": "severity", "value": "critical"},
		map[string]any{"key": "team", "value": "platform"},
	}
	if !labelsSetEqual(a, b) {
		t.Error("expected labels with same content in different order to be equal")
	}
}

func TestLabelsSetEqual_IgnoresIDField(t *testing.T) {
	// Old state has "id" field, new config does not
	a := []any{
		map[string]any{"id": "severity", "key": "severity", "value": "critical"},
	}
	b := []any{
		map[string]any{"key": "severity", "value": "critical"},
	}
	if !labelsSetEqual(a, b) {
		t.Error("expected labels to be equal even when old state has 'id' field")
	}
}

func TestLabelsSetEqual_DifferentValues(t *testing.T) {
	a := []any{
		map[string]any{"key": "severity", "value": "critical"},
	}
	b := []any{
		map[string]any{"key": "severity", "value": "warning"},
	}
	if labelsSetEqual(a, b) {
		t.Error("expected labels with different values to NOT be equal")
	}
}

func TestLabelsSetEqual_NilElementInFirstList(t *testing.T) {
	a := []any{
		nil,
		map[string]any{"key": "severity", "value": "critical"},
	}
	b := []any{
		map[string]any{"key": "severity", "value": "critical"},
		map[string]any{"key": "team", "value": "platform"},
	}
	if labelsSetEqual(a, b) {
		t.Error("expected false when first list contains a nil element")
	}
}

func TestLabelsSetEqual_NilElementInSecondList(t *testing.T) {
	a := []any{
		map[string]any{"key": "severity", "value": "critical"},
		map[string]any{"key": "team", "value": "platform"},
	}
	b := []any{
		nil,
		map[string]any{"key": "severity", "value": "critical"},
	}
	if labelsSetEqual(a, b) {
		t.Error("expected false when second list contains a nil element")
	}
}

func TestLabelsSetEqual_BothListsAllNil(t *testing.T) {
	a := []any{nil, nil}
	b := []any{nil, nil}
	if labelsSetEqual(a, b) {
		t.Error("expected false when both lists contain only nil elements")
	}
}

func TestLabelsSetEqual_DifferentCount(t *testing.T) {
	a := []any{
		map[string]any{"key": "severity", "value": "critical"},
	}
	b := []any{
		map[string]any{"key": "severity", "value": "critical"},
		map[string]any{"key": "team", "value": "platform"},
	}
	if labelsSetEqual(a, b) {
		t.Error("expected labels with different count to NOT be equal")
	}
}

func TestUnitValidateLabelKeyName(t *testing.T) {
	for _, tc := range []struct {
		key   string
		valid bool
	}{
		{key: "team", valid: true},
		{key: "a", valid: true},
		{key: "service_name", valid: true},
		{key: "region2region", valid: true},
		{key: strings.Repeat("k", labelNameMaxLength), valid: true},
		{key: strings.Repeat("k", labelNameMaxLength+1)},
		{key: ""},
		{key: "1Key"},
		{key: "Key1"},
		{key: "_key"},
		{key: "key_"},
		{key: "Key-Here"},
		{key: "Key.Here"},
		{key: "key name"},
	} {
		t.Run(tc.key, func(t *testing.T) {
			err := validateLabelKeyName(tc.key)
			if tc.valid && err != nil {
				t.Fatalf("expected %q to be valid, got %q", tc.key, err)
			}
			if !tc.valid && err == nil {
				t.Fatalf("expected %q to be rejected", tc.key)
			}
		})
	}
}

func TestUnitValidateLabelValueName(t *testing.T) {
	for _, tc := range []struct {
		value string
		valid bool
	}{
		{value: "critical", valid: true},
		{value: "prod-eu-west-2", valid: true},
		{value: "a1", valid: true},
		{value: "a", valid: true},
		{value: "release.1.2", valid: true},
		{value: "my_value", valid: true},
		{value: strings.Repeat("v", labelNameMaxLength), valid: true},
		{value: strings.Repeat("v", labelNameMaxLength+1)},
		{value: ""},
		{value: "1value"},
		{value: "-value"},
		{value: "_value"},
		{value: "value."},
		{value: "value-"},
		{value: "value with spaces"},
		{value: "{{ payload.severity }}"},
	} {
		t.Run(tc.value, func(t *testing.T) {
			err := validateLabelValueName(tc.value)
			if tc.valid && err != nil {
				t.Fatalf("expected %q to be valid, got %q", tc.value, err)
			}
			if !tc.valid && err == nil {
				t.Fatalf("expected %q to be rejected", tc.value)
			}
		})
	}
}

// labelsConfig builds the raw config value that Terraform passes to
// CustomizeDiff, with both label attributes present and null unless the test
// sets them.
func labelsConfig(attrs map[string]cty.Value) cty.Value {
	config := map[string]cty.Value{
		"labels":         cty.NullVal(cty.List(cty.Map(cty.String))),
		"dynamic_labels": cty.NullVal(cty.List(cty.Map(cty.String))),
	}
	for k, v := range attrs {
		config[k] = v
	}

	return cty.ObjectVal(config)
}

func labelValue(pairs map[string]string) cty.Value {
	label := make(map[string]cty.Value, len(pairs))
	for k, v := range pairs {
		label[k] = cty.StringVal(v)
	}

	return cty.MapVal(label)
}

func TestUnitValidateIntegrationLabelsConfig(t *testing.T) {
	for _, tc := range []struct {
		name         string
		config       cty.Value
		wantProblems []string
	}{
		{
			name: "valid static and dynamic labels",
			config: labelsConfig(map[string]cty.Value{
				"labels": cty.ListVal([]cty.Value{
					labelValue(map[string]string{"key": "region", "value": "prod-eu-west-2"}),
					labelValue(map[string]string{"id": "team", "key": "team", "value": "platform"}),
				}),
				"dynamic_labels": cty.ListVal([]cty.Value{
					labelValue(map[string]string{"key": "severity", "value": "{{ payload.get('severity', 'unknown') }}"}),
				}),
			}),
		},
		{
			name:   "no labels set",
			config: labelsConfig(nil),
		},
		{
			name: "empty label lists",
			config: labelsConfig(map[string]cty.Value{
				"labels":         cty.ListValEmpty(cty.Map(cty.String)),
				"dynamic_labels": cty.ListValEmpty(cty.Map(cty.String)),
			}),
		},
		{
			name: "static label value starting with a digit",
			config: labelsConfig(map[string]cty.Value{
				"labels": cty.ListVal([]cty.Value{
					labelValue(map[string]string{"key": "region", "value": "1prod"}),
				}),
			}),
			wantProblems: []string{`labels[0].value "1prod"`},
		},
		{
			name: "dynamic label key rejected but its template value accepted",
			config: labelsConfig(map[string]cty.Value{
				"dynamic_labels": cty.ListVal([]cty.Value{
					labelValue(map[string]string{"key": "1severity", "value": "{{ payload.severity }}"}),
				}),
			}),
			wantProblems: []string{`dynamic_labels[0].key "1severity"`},
		},
		{
			name: "every violation reported in config order",
			config: labelsConfig(map[string]cty.Value{
				"labels": cty.ListVal([]cty.Value{
					labelValue(map[string]string{"key": "region", "value": "1prod"}),
					labelValue(map[string]string{"key": "team", "value": "platform"}),
					labelValue(map[string]string{"key": "Key-Here", "value": "value."}),
				}),
				"dynamic_labels": cty.ListVal([]cty.Value{
					labelValue(map[string]string{"key": "1severity", "value": "{{ payload.severity }}"}),
				}),
			}),
			wantProblems: []string{
				`labels[0].value "1prod"`,
				`labels[2].key "Key-Here"`,
				`labels[2].value "value."`,
				`dynamic_labels[0].key "1severity"`,
			},
		},
		{
			name: "unresolved label list",
			config: labelsConfig(map[string]cty.Value{
				"labels": cty.UnknownVal(cty.List(cty.Map(cty.String))),
			}),
		},
		{
			name: "unresolved label element",
			config: labelsConfig(map[string]cty.Value{
				"labels": cty.ListVal([]cty.Value{cty.UnknownVal(cty.Map(cty.String))}),
			}),
		},
		{
			name: "unresolved label value",
			config: labelsConfig(map[string]cty.Value{
				"labels": cty.ListVal([]cty.Value{
					cty.MapVal(map[string]cty.Value{"key": cty.StringVal("region"), "value": cty.UnknownVal(cty.String)}),
				}),
			}),
		},
		{
			name:   "null config",
			config: cty.NullVal(cty.EmptyObject),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := validateIntegrationLabelsConfig(tc.config)

			if len(tc.wantProblems) == 0 {
				if err != nil {
					t.Fatalf("expected no error, got %q", err)
				}
				return
			}

			if err == nil {
				t.Fatalf("expected %d problems, got no error", len(tc.wantProblems))
			}

			message := err.Error()
			offset := 0
			for _, problem := range tc.wantProblems {
				i := strings.Index(message[offset:], problem)
				if i < 0 {
					t.Fatalf("expected %q after position %d in:\n%s", problem, offset, message)
				}
				offset += i + len(problem)
			}
			if got := strings.Count(message, "\n  "); got != len(tc.wantProblems) {
				t.Fatalf("expected %d problems, got %d in:\n%s", len(tc.wantProblems), got, message)
			}
		})
	}
}
