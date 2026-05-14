package frontendo11y

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestOrderIgnoredListModifier_SameElementsDifferentOrder(t *testing.T) {
	ctx := context.Background()

	stateValue, _ := types.ListValueFrom(ctx, types.StringType, []string{"b.com", "a.com", "c.com"})
	planValue, _ := types.ListValueFrom(ctx, types.StringType, []string{"a.com", "c.com", "b.com"})

	req := planmodifier.ListRequest{
		StateValue: stateValue,
		PlanValue:  planValue,
	}
	resp := &planmodifier.ListResponse{PlanValue: planValue}

	orderIgnoredListModifier{}.PlanModifyList(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %s", resp.Diagnostics.Errors())
	}

	// Diff should be suppressed — plan value should equal state value
	if !resp.PlanValue.Equal(stateValue) {
		t.Errorf("expected plan value to equal state value (diff suppressed), got %s", resp.PlanValue)
	}
}

func TestOrderIgnoredListModifier_DifferentElements(t *testing.T) {
	ctx := context.Background()

	stateValue, _ := types.ListValueFrom(ctx, types.StringType, []string{"a.com", "b.com"})
	planValue, _ := types.ListValueFrom(ctx, types.StringType, []string{"a.com", "c.com"})

	req := planmodifier.ListRequest{
		StateValue: stateValue,
		PlanValue:  planValue,
	}
	resp := &planmodifier.ListResponse{PlanValue: planValue}

	orderIgnoredListModifier{}.PlanModifyList(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %s", resp.Diagnostics.Errors())
	}

	// Diff should NOT be suppressed — elements differ
	if !resp.PlanValue.Equal(planValue) {
		t.Errorf("expected plan value to remain unchanged, got %s", resp.PlanValue)
	}
}

func TestOrderIgnoredListModifier_DifferentLengths(t *testing.T) {
	ctx := context.Background()

	stateValue, _ := types.ListValueFrom(ctx, types.StringType, []string{"a.com", "b.com"})
	planValue, _ := types.ListValueFrom(ctx, types.StringType, []string{"a.com", "b.com", "c.com"})

	req := planmodifier.ListRequest{
		StateValue: stateValue,
		PlanValue:  planValue,
	}
	resp := &planmodifier.ListResponse{PlanValue: planValue}

	orderIgnoredListModifier{}.PlanModifyList(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %s", resp.Diagnostics.Errors())
	}

	// Diff should NOT be suppressed — different number of elements
	if !resp.PlanValue.Equal(planValue) {
		t.Errorf("expected plan value to remain unchanged, got %s", resp.PlanValue)
	}
}

func TestOrderIgnoredListModifier_NullState(t *testing.T) {
	ctx := context.Background()

	planValue, _ := types.ListValueFrom(ctx, types.StringType, []string{"a.com"})

	req := planmodifier.ListRequest{
		StateValue: types.ListNull(types.StringType),
		PlanValue:  planValue,
	}
	resp := &planmodifier.ListResponse{PlanValue: planValue}

	orderIgnoredListModifier{}.PlanModifyList(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %s", resp.Diagnostics.Errors())
	}

	// New resource — plan should be unchanged
	if !resp.PlanValue.Equal(planValue) {
		t.Errorf("expected plan value to remain unchanged for null state, got %s", resp.PlanValue)
	}
}

func TestOrderIgnoredListModifier_IdenticalOrder(t *testing.T) {
	ctx := context.Background()

	stateValue, _ := types.ListValueFrom(ctx, types.StringType, []string{"a.com", "b.com"})
	planValue, _ := types.ListValueFrom(ctx, types.StringType, []string{"a.com", "b.com"})

	req := planmodifier.ListRequest{
		StateValue: stateValue,
		PlanValue:  planValue,
	}
	resp := &planmodifier.ListResponse{PlanValue: planValue}

	orderIgnoredListModifier{}.PlanModifyList(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %s", resp.Diagnostics.Errors())
	}

	// Same order, same elements — no change needed
	if !resp.PlanValue.Equal(planValue) {
		t.Errorf("expected plan value to remain unchanged, got %s", resp.PlanValue)
	}
}
