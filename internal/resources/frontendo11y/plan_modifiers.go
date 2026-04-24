package frontendo11y

import (
	"context"
	"sort"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// orderIgnoredListModifier is a plan modifier that suppresses diffs when a list
// attribute contains the same elements but in a different order. This is useful
// when an API does not guarantee the order of returned elements.
type orderIgnoredListModifier struct{}

func (m orderIgnoredListModifier) Description(_ context.Context) string {
	return "Ignores element ordering when comparing list values."
}

func (m orderIgnoredListModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m orderIgnoredListModifier) PlanModifyList(ctx context.Context, req planmodifier.ListRequest, resp *planmodifier.ListResponse) {
	// If the state is null (new resource) or the plan is unknown, nothing to compare.
	if req.StateValue.IsNull() || req.PlanValue.IsUnknown() {
		return
	}

	var stateElems, planElems []types.String
	resp.Diagnostics.Append(req.StateValue.ElementsAs(ctx, &stateElems, false)...)
	resp.Diagnostics.Append(req.PlanValue.ElementsAs(ctx, &planElems, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if len(stateElems) != len(planElems) {
		return
	}

	stateSorted := make([]string, len(stateElems))
	planSorted := make([]string, len(planElems))
	for i, v := range stateElems {
		stateSorted[i] = v.ValueString()
	}
	for i, v := range planElems {
		planSorted[i] = v.ValueString()
	}
	sort.Strings(stateSorted)
	sort.Strings(planSorted)

	for i := range stateSorted {
		if stateSorted[i] != planSorted[i] {
			return
		}
	}

	// Same elements, different order — suppress the diff by using the plan value
	// (which reflects the config ordering) so no update is triggered.
	resp.PlanValue = req.StateValue
}
