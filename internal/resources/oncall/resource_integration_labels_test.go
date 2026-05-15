package oncall

import (
	"testing"

	onCallAPI "github.com/grafana/amixr-api-go-client"
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
