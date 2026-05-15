package slo

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// emptyListForNullConfig coerces a nil config value to an empty list in the
// plan, while leaving non-nil config values (empty list and populated)
// untouched.
//
// Background: the SLO API marshals optional list fields with `omitempty`,
// so a PUT body containing `"groupByLabels": []` round-trips to a GET
// response with the field absent — indistinguishable from a never-set
// field. The OpenAPI client decodes that absence as a nil slice, so the
// natural read result is a nil `types.List`. Meanwhile a user HCL value
// of `group_by_labels = []` plans as an empty list, and Terraform Core's
// plan-validity check forbids `plan != config` for non-nil config values
// (even with `Computed: true`). The only direction that satisfies the
// rule for both shapes is to make every config — nil, empty, populated —
// converge on a non-nil plan value, which is what this modifier does for
// the nil case. The backward (read) path mirrors by promoting nil API
// responses to an empty list, keeping the state shape stable.
//
// The schema must declare the attribute `Computed: true` so the framework
// permits the modifier to fill in a value when config is nil. When config
// is `[]` or populated, terraform-Core's rule (`plan == config`) holds
// trivially and the modifier is a no-op.
type emptyListForNullConfig struct{}

func EmptyListForNullConfig() planmodifier.List {
	return emptyListForNullConfig{}
}

func (emptyListForNullConfig) Description(_ context.Context) string {
	return "Coerces a nil config value to an empty list in the plan."
}

func (emptyListForNullConfig) MarkdownDescription(_ context.Context) string {
	return "Coerces a nil config value to an empty list (`[]`) in the plan."
}

func (emptyListForNullConfig) PlanModifyList(ctx context.Context, req planmodifier.ListRequest, resp *planmodifier.ListResponse) {
	if !req.ConfigValue.IsNull() {
		return
	}
	emptyList, diags := types.ListValueFrom(ctx, req.ConfigValue.ElementType(ctx), []string{})
	resp.Diagnostics.Append(diags...)
	resp.PlanValue = emptyList
}
