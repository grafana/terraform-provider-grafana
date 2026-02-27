package oncall

import (
	"testing"

	onCallAPI "github.com/grafana/amixr-api-go-client"
	"github.com/stretchr/testify/assert"
)

func TestFlattenLabels_SortedByKey(t *testing.T) {
	tests := []struct {
		name     string
		input    []*onCallAPI.Label
		expected []map[string]string
	}{
		{
			name:     "empty labels",
			input:    []*onCallAPI.Label{},
			expected: []map[string]string{},
		},
		{
			name: "single label",
			input: []*onCallAPI.Label{
				{Key: onCallAPI.KeyValueName{Name: "service"}, Value: onCallAPI.KeyValueName{Name: "web"}},
			},
			expected: []map[string]string{
				{"id": "service", "key": "service", "value": "web"},
			},
		},
		{
			name: "already sorted labels",
			input: []*onCallAPI.Label{
				{Key: onCallAPI.KeyValueName{Name: "alpha"}, Value: onCallAPI.KeyValueName{Name: "val1"}},
				{Key: onCallAPI.KeyValueName{Name: "beta"}, Value: onCallAPI.KeyValueName{Name: "val2"}},
				{Key: onCallAPI.KeyValueName{Name: "gamma"}, Value: onCallAPI.KeyValueName{Name: "val3"}},
			},
			expected: []map[string]string{
				{"id": "alpha", "key": "alpha", "value": "val1"},
				{"id": "beta", "key": "beta", "value": "val2"},
				{"id": "gamma", "key": "gamma", "value": "val3"},
			},
		},
		{
			name: "unsorted labels are returned sorted by key",
			input: []*onCallAPI.Label{
				{Key: onCallAPI.KeyValueName{Name: "service_name"}, Value: onCallAPI.KeyValueName{Name: "{{ payload.commonLabels.service_name }}"}},
				{Key: onCallAPI.KeyValueName{Name: "namespace"}, Value: onCallAPI.KeyValueName{Name: "{{ payload.commonLabels.namespace }}"}},
				{Key: onCallAPI.KeyValueName{Name: "severity"}, Value: onCallAPI.KeyValueName{Name: "{{ payload.commonLabels.severity }}"}},
				{Key: onCallAPI.KeyValueName{Name: "project"}, Value: onCallAPI.KeyValueName{Name: "{{ payload.commonLabels.project }}"}},
			},
			expected: []map[string]string{
				{"id": "namespace", "key": "namespace", "value": "{{ payload.commonLabels.namespace }}"},
				{"id": "project", "key": "project", "value": "{{ payload.commonLabels.project }}"},
				{"id": "service_name", "key": "service_name", "value": "{{ payload.commonLabels.service_name }}"},
				{"id": "severity", "key": "severity", "value": "{{ payload.commonLabels.severity }}"},
			},
		},
		{
			name: "reverse sorted labels are returned sorted by key",
			input: []*onCallAPI.Label{
				{Key: onCallAPI.KeyValueName{Name: "z_label"}, Value: onCallAPI.KeyValueName{Name: "last"}},
				{Key: onCallAPI.KeyValueName{Name: "m_label"}, Value: onCallAPI.KeyValueName{Name: "middle"}},
				{Key: onCallAPI.KeyValueName{Name: "a_label"}, Value: onCallAPI.KeyValueName{Name: "first"}},
			},
			expected: []map[string]string{
				{"id": "a_label", "key": "a_label", "value": "first"},
				{"id": "m_label", "key": "m_label", "value": "middle"},
				{"id": "z_label", "key": "z_label", "value": "last"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := flattenLabels(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
