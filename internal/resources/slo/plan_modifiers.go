package slo

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// emptyListAsNull collapses a known, empty plan list to null.
//
// The SLO API marshals optional list fields with `omitempty`, so a PUT body
// containing `"groupByLabels": []` round-trips to a GET response with the
// field absent, which the OpenAPI client decodes as a nil slice — i.e.
// indistinguishable from a never-set field. Treating config `[]` as null on
// the plan side keeps state and config aligned regardless of whether the user
// wrote `group_by_labels = []` or omitted the attribute, avoiding the
// "was cty.ListValEmpty(cty.String), but now null" inconsistency error.
type emptyListAsNull struct{}

func EmptyListAsNull() planmodifier.List {
	return emptyListAsNull{}
}

func (emptyListAsNull) Description(_ context.Context) string {
	return "Treats an empty list value (`[]`) the same as null."
}

func (emptyListAsNull) MarkdownDescription(ctx context.Context) string {
	return "Treats an empty list value (`[]`) the same as null."
}

func (emptyListAsNull) PlanModifyList(ctx context.Context, req planmodifier.ListRequest, resp *planmodifier.ListResponse) {
	if req.PlanValue.IsNull() || req.PlanValue.IsUnknown() {
		return
	}
	if len(req.PlanValue.Elements()) == 0 {
		resp.PlanValue = types.ListNull(req.PlanValue.ElementType(ctx))
	}
}
