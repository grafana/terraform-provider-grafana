package fleetmanagement

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/stretchr/testify/assert"
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
			assert.Equal(t, tt.expected, actual)
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
			assert.Equal(t, tt.expected, actual)
		})
	}
}

func TestNativeSliceToTFList(t *testing.T) {
	tests := []struct {
		name        string
		nativeSlice []string
		expected    types.List
	}{
		{
			"nil slice",
			nil,
			types.ListValueMust(types.StringType, []attr.Value{}),
		},
		{
			"empty slice",
			[]string{},
			types.ListValueMust(types.StringType, []attr.Value{}),
		},
		{
			"non-empty slice",
			[]string{"a", "b"},
			types.ListValueMust(types.StringType,
				[]attr.Value{
					types.StringValue("a"),
					types.StringValue("b"),
				},
			),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := nativeSliceToTFList(tt.nativeSlice)
			assert.Equal(t, tt.expected, actual)
		})
	}
}

func TestTfListToNativeSlice(t *testing.T) {
	tests := []struct {
		name     string
		tfList   types.List
		expected []string
		err      error
	}{
		{
			"null list",
			basetypes.NewListNull(types.StringType),
			[]string{},
			nil,
		},
		{
			"unknown list",
			basetypes.NewListUnknown(types.StringType),
			[]string{},
			nil,
		},
		{
			"empty list",
			types.ListValueMust(types.StringType, []attr.Value{}),
			[]string{},
			nil,
		},
		{
			"non-empty list",
			types.ListValueMust(types.StringType,
				[]attr.Value{
					types.StringValue("a"),
					types.StringValue("b"),
				},
			),
			[]string{"a", "b"},
			nil,
		},
		{
			"invalid list type",
			types.ListValueMust(types.BoolType,
				[]attr.Value{
					types.BoolValue(true),
				},
			),
			nil,
			fmt.Errorf("unexpected type for element at index 0: expected types.String, got basetypes.BoolValue"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual, err := tfListToNativeSlice(tt.tfList)
			assert.Equal(t, tt.expected, actual)
			assert.Equal(t, tt.err, err)
		})
	}
}

func TestNativeMapToTFMap(t *testing.T) {
	tests := []struct {
		name      string
		nativeMap map[string]string
		expected  types.Map
	}{
		{
			"nil map",
			nil,
			types.MapValueMust(
				types.StringType,
				map[string]attr.Value{},
			),
		},
		{
			"empty map",
			map[string]string{},
			types.MapValueMust(
				types.StringType,
				map[string]attr.Value{},
			),
		},
		{
			"non-empty map",
			map[string]string{
				"key1": "value1",
				"key2": "value2",
			},
			types.MapValueMust(
				types.StringType,
				map[string]attr.Value{
					"key1": types.StringValue("value1"),
					"key2": types.StringValue("value2"),
				},
			),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := nativeMapToTFMap(tt.nativeMap)
			assert.Equal(t, tt.expected, actual)
		})
	}
}

func TestTfMapToNativeMap(t *testing.T) {
	tests := []struct {
		name     string
		tfMap    types.Map
		expected map[string]string
		err      error
	}{
		{
			"null map",
			basetypes.NewMapNull(types.StringType),
			map[string]string{},
			nil,
		},
		{
			"unknown map",
			basetypes.NewMapUnknown(types.StringType),
			map[string]string{},
			nil,
		},
		{
			"empty map",
			types.MapValueMust(
				types.StringType,
				map[string]attr.Value{},
			),
			map[string]string{},
			nil,
		},
		{
			"non-empty map",
			types.MapValueMust(
				types.StringType,
				map[string]attr.Value{
					"key1": types.StringValue("value1"),
					"key2": types.StringValue("value2"),
				},
			),
			map[string]string{
				"key1": "value1",
				"key2": "value2",
			},
			nil,
		},
		{
			"invalid map type",
			types.MapValueMust(
				types.BoolType,
				map[string]attr.Value{
					"key1": types.BoolValue(true),
				},
			),
			nil,
			fmt.Errorf("unexpected type for key \"key1\": expected types.String, got basetypes.BoolValue"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual, err := tfMapToNativeMap(tt.tfMap)
			assert.Equal(t, tt.expected, actual)
			assert.Equal(t, tt.err, err)
		})
	}
}
