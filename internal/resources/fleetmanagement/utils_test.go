package fleetmanagement

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/require"
)

func TestTfBoolToNativeBoolPtr(t *testing.T) {
	truev := true
	falsev := false

	tests := []struct {
		name     string
		tfBool   types.Bool
		expected *bool
	}{
		{
			"null bool",
			types.BoolNull(),
			nil,
		},
		{
			"unknown bool",
			types.BoolUnknown(),
			nil,
		},
		{
			"true bool",
			types.BoolValue(true),
			&truev,
		},
		{
			"false bool",
			types.BoolValue(false),
			&falsev,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := tfBoolToNativeBoolPtr(tt.tfBool)
			require.Equal(t, tt.expected, actual)
		})
	}
}

func TestTfStringToNativeStringPtr(t *testing.T) {
	testStr := "test"

	tests := []struct {
		name     string
		tfString types.String
		expected *string
	}{
		{
			"null string",
			types.StringNull(),
			nil,
		},
		{
			"unknown string",
			types.StringUnknown(),
			nil,
		},
		{
			"non-empty string",
			types.StringValue(testStr),
			&testStr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := tfStringToNativeStringPtr(tt.tfString)
			require.Equal(t, tt.expected, actual)
		})
	}
}
