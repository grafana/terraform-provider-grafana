package assistant

import (
	"context"
	"testing"
	"time"

	"github.com/grafana/terraform-provider-grafana/v4/internal/common/assistantapi"
)

func TestUnitSkillToModelCommandState(t *testing.T) {
	t.Parallel()

	commandName := "deploy"
	enabledAt := time.Now()

	enabled, diags := skillToModel(context.Background(), assistantapi.Skill{
		CommandName:      &commandName,
		CommandEnabledAt: &enabledAt,
	})
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if enabled.CommandName.ValueString() != commandName {
		t.Fatalf("unexpected enabled command name: %q", enabled.CommandName.ValueString())
	}

	disabled, diags := skillToModel(context.Background(), assistantapi.Skill{
		CommandName: &commandName,
	})
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if !disabled.CommandName.IsNull() {
		t.Fatalf("expected disabled command name to be null, got %q", disabled.CommandName.ValueString())
	}
}
