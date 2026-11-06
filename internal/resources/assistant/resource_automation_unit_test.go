package assistant

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/grafana/terraform-provider-grafana/v4/internal/common/assistantapi"
)

// Keep a concrete conversion result even for an absent block. The sparse
// update builder compares individual destinations and emits null only for
// destinations that the configuration removed.
func TestUnitAutomationNotificationsFromModelIsNeverNil(t *testing.T) {
	t.Parallel()

	got, diags := automationNotificationsFromModel(context.Background(), nil)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if got == nil {
		t.Fatal("nil provider set would leave stale notifications in place")
	}
	if got.Slack != nil || got.Email != nil {
		t.Fatalf("expected both providers nil so they serialize as null, got %+v", got)
	}
}

func TestUnitAutomationScheduleRoundTrip(t *testing.T) {
	t.Parallel()

	model, diags := automationToModel(context.Background(), assistantapi.Automation{
		ID:      "a1",
		Name:    "daily",
		Prompt:  "summarize",
		Enabled: true,
		Scope:   "tenant",
		Schedule: &assistantapi.AutomationSchedule{
			Cron:      "0 9 * * *",
			Timezone:  "Europe/Paris",
			NextRunAt: "2026-08-04T09:00:00Z",
		},
	}, automationModel{})
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if model.ScheduleCron.ValueString() != "0 9 * * *" {
		t.Errorf("cron = %q", model.ScheduleCron.ValueString())
	}
	if model.ScheduleTimezone.ValueString() != "Europe/Paris" {
		t.Errorf("timezone = %q", model.ScheduleTimezone.ValueString())
	}
	if model.NextRunAt.ValueString() != "2026-08-04T09:00:00Z" {
		t.Errorf("next_run_at = %q", model.NextRunAt.ValueString())
	}
}

// A manual-only automation has no schedule at all; the mapped attributes must
// stay null rather than becoming empty strings.
func TestUnitAutomationWithoutScheduleLeavesAttributesNull(t *testing.T) {
	t.Parallel()

	model, diags := automationToModel(context.Background(), assistantapi.Automation{
		ID: "a1", Name: "manual", Prompt: "p", Scope: "user",
	}, automationModel{})
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if !model.ScheduleCron.IsNull() || !model.ScheduleTimezone.IsNull() || !model.NextRunAt.IsNull() {
		t.Fatalf("expected null schedule attributes, got %q %q %q",
			model.ScheduleCron, model.ScheduleTimezone, model.NextRunAt)
	}
}

func TestUnitAutomationNotificationsToModelIgnoresUnconfiguredDefaults(t *testing.T) {
	t.Parallel()

	got, diags := automationNotificationsToModel(context.Background(), &assistantapi.AutomationNotifications{
		Slack: &assistantapi.AutomationSlackNotification{Enabled: false},
		Email: &assistantapi.AutomationEmailNotification{Enabled: false},
	}, nil)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if got != nil {
		t.Fatalf("expected nil notifications for an unconfigured set, got %+v", got)
	}
}

func TestUnitAutomationSlackNotificationRoundTrip(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	notifyOn, diags := types.ListValueFrom(ctx, types.StringType, []string{"completed", "failed"})
	if diags.HasError() {
		t.Fatalf("building notify_on: %v", diags)
	}

	cfg := &automationNotificationsModel{Slack: &automationSlackModel{
		Enabled:  types.BoolValue(true),
		NotifyOn: notifyOn,
		Target: &automationSlackTargetModel{
			Type:      types.StringValue("channel"),
			ChannelID: types.StringValue("C0BJTU4CLGK"),
		},
	}}

	sent, diags := automationNotificationsFromModel(ctx, cfg)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if sent.Slack == nil || sent.Slack.Target == nil || sent.Slack.Target.ChannelID != "C0BJTU4CLGK" {
		t.Fatalf("slack target lost on the way out: %+v", sent.Slack)
	}
	if len(sent.Slack.NotifyOn) != 2 {
		t.Fatalf("notify_on lost: %+v", sent.Slack.NotifyOn)
	}
	if sent.Email != nil {
		t.Fatalf("unconfigured email must stay nil so it serializes as null, got %+v", sent.Email)
	}

	back, diags := automationNotificationsToModel(ctx, sent, cfg)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if back == nil || back.Slack == nil || back.Slack.Target == nil {
		t.Fatalf("slack target lost on the way back: %+v", back)
	}
	if back.Slack.Target.ChannelID.ValueString() != "C0BJTU4CLGK" {
		t.Errorf("channel id = %q", back.Slack.Target.ChannelID.ValueString())
	}
}

func TestUnitAutomationNotificationNormalizationAndDrift(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	for _, tc := range []struct {
		name       string
		configured []string
		remote     []string
		email      bool
		equivalent bool
	}{
		{"outcome order and duplicates", []string{"failed", "completed", "failed"}, []string{"completed", "failed"}, false, true},
		{"changed outcomes", []string{"completed"}, []string{"failed"}, false, false},
		{"email canonical spelling", []string{" A@example.invalid ", "a@example.invalid", ""}, []string{"A@example.invalid"}, true, true},
		{"changed recipient", []string{"a@example.invalid"}, []string{"b@example.invalid"}, true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			configured, diags := types.ListValueFrom(ctx, types.StringType, tc.configured)
			if diags.HasError() {
				t.Fatal(diags)
			}
			got, diags := automationNotificationListToModel(ctx, tc.remote, configured, tc.email)
			if diags.HasError() {
				t.Fatal(diags)
			}
			if got.Equal(configured) != tc.equivalent {
				t.Fatalf("equivalent = %t, got %s for configured %s", tc.equivalent, got, configured)
			}
		})
	}
}

func TestUnitAutomationResolvedSlackTargetOwnership(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	remote := &assistantapi.AutomationNotifications{Slack: &assistantapi.AutomationSlackNotification{
		Enabled: true,
		Target:  &assistantapi.AutomationSlackTarget{Type: "dm", UserID: "U123", TeamID: "T123"},
	}}
	configured := &automationNotificationsModel{Slack: &automationSlackModel{
		Enabled: types.BoolValue(true), NotifyOn: types.ListNull(types.StringType),
		Target: &automationSlackTargetModel{Type: types.StringValue("dm"), UserID: types.StringNull(), TeamID: types.StringNull()},
	}}
	got, diags := automationNotificationsToModel(ctx, remote, configured)
	if diags.HasError() {
		t.Fatal(diags)
	}
	if !got.Slack.Target.UserID.IsNull() || !got.Slack.Target.TeamID.IsNull() {
		t.Fatalf("backend-resolved IDs must not become configured inputs: %+v", got.Slack.Target)
	}
	imported, diags := automationNotificationsToModel(ctx, remote, nil)
	if diags.HasError() {
		t.Fatal(diags)
	}
	if imported.Slack.Target.UserID.ValueString() != "U123" || imported.Slack.Target.TeamID.ValueString() != "T123" {
		t.Fatalf("import lost saved IDs: %+v", imported.Slack.Target)
	}
	configured.Slack.Target.TeamID = types.StringValue("T-other")
	got, diags = automationNotificationsToModel(ctx, remote, configured)
	if diags.HasError() || got.Slack.Target.TeamID.ValueString() != "T123" {
		t.Fatalf("configured ID drift must remain visible: %+v, %v", got, diags)
	}
}
