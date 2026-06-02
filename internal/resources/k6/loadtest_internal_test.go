package k6

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestUnitHandleK6Version(t *testing.T) {
	ptr := func(i int32) *int32 { return &i }

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
			// Regression: 0 is a valid version id and must not be treated as null.
			name:     "zero is preserved",
			value:    ptr(0),
			expected: types.StringValue("0"),
		},
		{
			name:     "positive value is preserved",
			value:    ptr(42),
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
