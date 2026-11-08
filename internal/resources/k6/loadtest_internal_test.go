package k6

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
)

func TestUnitHandleK6Version(t *testing.T) {
	for _, tc := range []struct {
		name     string
		value    *int32
		expected types.String
	}{
		{
			name:     "null or absent is null",
			value:    nil,
			expected: types.StringNull(),
		},
		{
			name:     "zero is preserved",
			value:    common.Ref[int32](0),
			expected: types.StringValue("0"),
		},
		{
			name:     "positive value is preserved",
			value:    common.Ref[int32](42),
			expected: types.StringValue("42"),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := handleK6Version(tc.value)
			if !got.Equal(tc.expected) {
				t.Errorf("handleK6Version(%v) = %v, want %v", tc.value, got, tc.expected)
			}
		})
	}
}
