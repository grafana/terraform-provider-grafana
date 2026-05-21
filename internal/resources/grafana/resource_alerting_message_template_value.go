package grafana

import (
	"context"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

var _ basetypes.StringValuableWithSemanticEquals = templateValue{}

type templateValue struct {
	basetypes.StringValue
}

func (v templateValue) Normalized() string {
	return strings.TrimSpace(v.StringValue.ValueString())
}

func (v templateValue) Equal(o attr.Value) bool {
	other, ok := o.(templateValue)
	if !ok {
		return false
	}
	return v.StringValue.Equal(other.StringValue)
}

func (v templateValue) Type(context.Context) attr.Type {
	return templateType{}
}

func (v templateValue) StringSemanticEquals(_ context.Context, newValuable basetypes.StringValuable) (bool, diag.Diagnostics) {
	var diags diag.Diagnostics

	other, ok := newValuable.(templateValue)
	if !ok {
		diags.AddError(
			"Semantic Equality Check Error",
			"unexpected value type for template semantic equality",
		)
		return false, diags
	}

	return v.Normalized() == other.Normalized(), diags
}
