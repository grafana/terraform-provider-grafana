package fleetmanagement

import (
	"bytes"
	"context"
	"fmt"

	"github.com/grafana/river/parser"
	"github.com/grafana/river/printer"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/attr/xattr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

var (
	_ basetypes.StringValuable                   = AlloyConfigValue{}
	_ basetypes.StringValuableWithSemanticEquals = AlloyConfigValue{}
	_ xattr.ValidateableAttribute                = AlloyConfigValue{}
)

type AlloyConfigValue struct {
	basetypes.StringValue
}

func NewAlloyConfigValue(value string) AlloyConfigValue {
	return AlloyConfigValue{
		StringValue: basetypes.NewStringValue(value),
	}
}

func (v AlloyConfigValue) Equal(o attr.Value) bool {
	other, ok := o.(AlloyConfigValue)
	if !ok {
		return false
	}

	return v.StringValue.Equal(other.StringValue)
}

func (v AlloyConfigValue) Type(ctx context.Context) attr.Type {
	return AlloyConfigType{}
}

func (v AlloyConfigValue) StringSemanticEquals(ctx context.Context, newValuable basetypes.StringValuable) (bool, diag.Diagnostics) {
	var diags diag.Diagnostics

	newValue, ok := newValuable.(AlloyConfigValue)
	if !ok {
		diags.AddError(
			"Semantic Equality Check Error",
			"An unexpected value type was received while performing semantic equality checks. "+
				"Please report this to the provider developers.\n\n"+
				"Expected Value Type: "+fmt.Sprintf("%T", v)+"\n"+
				"Got Value Type: "+fmt.Sprintf("%T", newValuable),
		)

		return false, diags
	}

	result, err := riverEqual(v.ValueString(), newValue.ValueString())
	if err != nil {
		diags.AddError(
			"Semantic Equality Check Error",
			"An unexpected error occurred while performing semantic equality checks. "+
				"Please report this to the provider developers.\n\n"+
				"Error: "+err.Error(),
		)

		return false, diags
	}

	return result, diags
}

func (v AlloyConfigValue) ValidateAttribute(ctx context.Context, req xattr.ValidateAttributeRequest, resp *xattr.ValidateAttributeResponse) {
	if v.IsNull() || v.IsUnknown() {
		return
	}

	_, err := parseRiver(v.ValueString())
	if err != nil {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid Alloy configuration",
			"An unexpected error occurred while parsing Alloy configuration. "+
				"Path: "+req.Path.String()+"\n"+
				"Given Value: "+v.ValueString()+"\n"+
				"Error: "+err.Error(),
		)

		return
	}
}

func riverEqual(contents1 string, contents2 string) (bool, error) {
	parsed1, err := parseRiver(contents1)
	if err != nil {
		return false, err
	}

	parsed2, err := parseRiver(contents2)
	if err != nil {
		return false, err
	}

	return parsed1 == parsed2, nil
}

func parseRiver(contents string) (string, error) {
	file, err := parser.ParseFile("", []byte(contents))
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := printer.Fprint(&buf, file); err != nil {
		return "", err
	}

	return buf.String(), nil
}
