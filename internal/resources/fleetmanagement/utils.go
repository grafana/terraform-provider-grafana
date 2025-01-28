package fleetmanagement

import (
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
