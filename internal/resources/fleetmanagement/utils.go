package fleetmanagement

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func tfBoolToNativeBoolPtr(tfBool types.Bool) *bool {
	var boolPtr *bool
	if !(tfBool.IsNull() || tfBool.IsUnknown()) {
		val := tfBool.ValueBool()
		boolPtr = &val
	}
	return boolPtr
}

func tfStringToNativeStringPtr(tfString types.String) *string {
	var stringPtr *string
	if !(tfString.IsNull() || tfString.IsUnknown()) {
		val := tfString.ValueString()
		stringPtr = &val
	}
	return stringPtr
}

func nativeSliceToTFList(nativeSlice []string) types.List {
	tfList := make([]attr.Value, len(nativeSlice))
	for i, elem := range nativeSlice {
		tfList[i] = types.StringValue(elem)
	}
	return types.ListValueMust(types.StringType, tfList)
}

func tfListToNativeSlice(tfList types.List) ([]string, error) {
	elements := tfList.Elements()
	nativeSlice := make([]string, len(elements))
	for i, elem := range elements {
		valStr, ok := elem.(types.String)
		if !ok {
			return nil, fmt.Errorf("unexpected type for element at index %d: expected types.String, got %T", i, elem)
		}
		nativeSlice[i] = valStr.ValueString()
	}
	return nativeSlice, nil
}

func nativeMapToTFMap(nativeMap map[string]string) types.Map {
	tfMap := make(map[string]attr.Value, len(nativeMap))
	for key, val := range nativeMap {
		tfMap[key] = types.StringValue(val)
	}
	return types.MapValueMust(types.StringType, tfMap)
}

func tfMapToNativeMap(tfMap types.Map) (map[string]string, error) {
	elements := tfMap.Elements()
	nativeMap := make(map[string]string, len(elements))
	for key, val := range elements {
		valStr, ok := val.(types.String)
		if !ok {
			return nil, fmt.Errorf("unexpected type for key %q: expected types.String, got %T", key, val)
		}
		nativeMap[key] = valStr.ValueString()
	}
	return nativeMap, nil
}
