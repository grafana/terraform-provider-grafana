package k6

import (
	"encoding/json"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/grafana/k6-cloud-openapi-client-go/k6"
)

func TestUnitHandleGrafanaFolderUID(t *testing.T) {
	folderUID := "folder-uid"
	var explicitNull k6.NullableString
	if err := json.Unmarshal([]byte("null"), &explicitNull); err != nil {
		t.Fatalf("unmarshal explicit null: %v", err)
	}

	for _, tc := range []struct {
		name     string
		value    k6.NullableString
		expected types.String
	}{
		{
			name:     "unset is null",
			value:    k6.NullableString{},
			expected: types.StringNull(),
		},
		{
			name:     "explicit null is null",
			value:    explicitNull,
			expected: types.StringNull(),
		},
		{
			name:     "value is preserved",
			value:    *k6.NewNullableString(&folderUID),
			expected: types.StringValue(folderUID),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := handleGrafanaFolderUID(tc.value)
			if !got.Equal(tc.expected) {
				t.Errorf("handleGrafanaFolderUID(%v) = %v, want %v", tc.value, got, tc.expected)
			}
		})
	}
}
