package appplatform

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/require"
)

func TestAttributeCountExactlyWithUnknownValues(t *testing.T) {
	set, null, unknown := types.StringValue("x"), types.StringNull(), types.StringUnknown()

	tests := []struct {
		name    string
		a, b, c attr.Value
		wantErr bool
	}{
		{name: "one known", a: set, b: null, c: null},
		{name: "none set", a: null, b: null, c: null, wantErr: true},
		{name: "two known", a: set, b: set, c: null, wantErr: true},
		{name: "one unknown could be the one", a: unknown, b: null, c: null},
		{name: "all unknown could be exactly one", a: unknown, b: unknown, c: unknown},
		{name: "one known and one unknown could be exactly one", a: set, b: unknown, c: null},
		{name: "two known and one unknown is too many", a: set, b: set, c: unknown, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := types.ObjectValueMust(
				map[string]attr.Type{"a": types.StringType, "b": types.StringType, "c": types.StringType},
				map[string]attr.Value{"a": tt.a, "b": tt.b, "c": tt.c},
			)
			resp := &validator.ObjectResponse{}
			attributeCountExactly(1, "a", "b", "c").ValidateObject(context.Background(), validator.ObjectRequest{
				Path:        path.Root("if"),
				ConfigValue: config,
			}, resp)
			require.Equal(t, tt.wantErr, resp.Diagnostics.HasError(), "%v", resp.Diagnostics)
		})
	}
}

func TestMatcherValidatorWithUnknownValues(t *testing.T) {
	matcher := func(matchType, name, value types.String) attr.Value {
		return types.ObjectValueMust(matcherType.AttrTypes, map[string]attr.Value{
			"type": matchType, "name": name, "value": value,
		})
	}
	unknown := types.StringUnknown()
	valid := matcher(types.StringValue("="), types.StringValue("a"), types.StringValue("b"))
	emptyName := matcher(types.StringValue("="), types.StringValue(""), types.StringValue("b"))
	badType := matcher(types.StringValue("bad"), types.StringValue("a"), types.StringValue("b"))
	unknownElement := types.ObjectUnknown(matcherType.AttrTypes)

	tests := []struct {
		name     string
		elements []attr.Value
		wantErr  string
	}{
		{name: "known valid", elements: []attr.Value{valid}},
		{name: "known empty name", elements: []attr.Value{emptyName}, wantErr: "index 0 is invalid: matcher 'type' and 'name' must be set"},
		{name: "known bad type", elements: []attr.Value{badType}, wantErr: "index 0 is invalid: invalid matcher type"},
		{name: "unknown type", elements: []attr.Value{matcher(unknown, types.StringValue("a"), types.StringValue("b"))}},
		{name: "unknown name", elements: []attr.Value{matcher(types.StringValue("="), unknown, types.StringValue("b"))}},
		{name: "unknown value", elements: []attr.Value{matcher(types.StringValue("="), types.StringValue("a"), unknown)}},
		{name: "unknown element", elements: []attr.Value{unknownElement}},
		{name: "unknown element does not hide a bad one", elements: []attr.Value{unknownElement, badType}, wantErr: "index 1 is invalid: invalid matcher type"},
		{name: "unknown name does not hide a bad type", elements: []attr.Value{matcher(types.StringValue("bad"), unknown, types.StringValue("b"))}, wantErr: "index 0 is invalid: invalid matcher type"},
		{name: "unknown type does not hide an empty name", elements: []attr.Value{matcher(unknown, types.StringValue(""), types.StringValue("b"))}, wantErr: "index 0 is invalid: matcher 'type' and 'name' must be set"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := &validator.ListResponse{}
			matcherValidator{}.ValidateList(context.Background(), validator.ListRequest{
				Path:        path.Root("label_matchers"),
				ConfigValue: types.ListValueMust(matcherType, tt.elements),
			}, resp)
			if tt.wantErr == "" {
				require.False(t, resp.Diagnostics.HasError(), "%v", resp.Diagnostics)
				return
			}
			require.Len(t, resp.Diagnostics.Errors(), 1, "%v", resp.Diagnostics)
			require.Contains(t, resp.Diagnostics.Errors()[0].Detail(), tt.wantErr)
		})
	}
}
